// Package services 提供房间服务的业务逻辑实现
package services

import (
	"bytes"                      // 字节缓冲区操作
	"context"                    // 上下文管理
	"crypto/tls"                 // TLS加密连接
	"crypto/x509"                // X.509证书处理
	"encoding/json"              // JSON编码解码
	"fmt"                        // 格式化输出
	"go-backEnd/internal/config" // 配置管理
	"go-backEnd/internal/models" // 数据模型
	"go-backEnd/pkg/audio"       // 音频处理包
	"log"                        // 日志记录
	"strings"                    // 字符串操作
	"time"                       // 时间处理

	"github.com/google/uuid"       // UUID生成
	"github.com/gorilla/websocket" // WebSocket连接
)

// RoomService 房间服务结构体，负责管理单个房间的所有业务逻辑
type RoomService struct {
	room   *models.Room       // 关联的房间对象指针
	ctx    context.Context    // 用于协程生命周期管理
	cancel context.CancelFunc // 取消函数，用于停止所有协程
}

// NewRoomService 创建新的房间服务实例
func NewRoomService(room *models.Room) *RoomService {
	ctx, cancel := context.WithCancel(context.Background()) // 创建可取消的上下文

	// 注册房间到监控系统
	monitor := GetTranslationMonitor()
	monitor.RegisterRoom(room.ID, "single", len(room.Clients))

	return &RoomService{
		room:   room,   // 关联的房间对象指针
		ctx:    ctx,    // 上下文对象
		cancel: cancel, // 取消函数
	}
}

// Run 房间服务主运行循环，处理所有房间相关的事件
func (rs *RoomService) Run() {
	monitor := GetTranslationMonitor()

	// 添加房间服务协程到监控
	monitor.AddGoroutine(rs.room.ID, "room_service")

	// 添加panic恢复机制，防止房间服务异常退出
	defer func() {
		if r := recover(); r != nil {
			log.Printf("❌ [ROOM %s] 服务异常恢复: %v", rs.room.ID, r)
		}
		// 移除协程监控和房间注册
		monitor.RemoveGoroutine(rs.room.ID, "room_service")
		monitor.UnregisterRoom(rs.room.ID)
		rs.cancel() // 确保退出时取消所有协程
	}()

	for { // 无限循环处理房间事件
		select { // 监听多个通道事件
		case <-rs.ctx.Done(): // 检查上下文是否被取消
			log.Printf("🛑 [ROOM %s] 服务正常退出", rs.room.ID)
			return
		case client := <-rs.room.Register: // 处理客户端注册事件
			rs.room.Clients[client] = true                            // 将客户端添加到房间客户端映射中
			rs.room.ClientAudioBuffers.Store(client, &bytes.Buffer{}) // 为客户端创建音频缓冲区
			rs.room.ShouldStopTrans = false                           // 设置翻译服务不应停止

			// 更新监控中的客户端数量
			monitor.UpdateClientCount(rs.room.ID, len(rs.room.Clients))

			if rs.room.TranslationWS == nil { // 如果翻译WebSocket连接不存在
				go rs.StartTranslationService() // 启动翻译服务协程
			}
		case client := <-rs.room.Unregister: // 处理客户端注销事件
			if _, ok := rs.room.Clients[client]; ok { // 检查客户端是否存在于房间中
				delete(rs.room.Clients, client)           // 从房间客户端映射中删除客户端
				close(client.Send)                        // 关闭客户端发送通道
				rs.room.ClientAudioBuffers.Delete(client) // 删除客户端音频缓冲区
			}

			// 更新监控中的客户端数量
			monitor.UpdateClientCount(rs.room.ID, len(rs.room.Clients))

			if len(rs.room.Clients) == 0 { // 如果房间没有客户端了
				rs.room.ShouldStopTrans = true                     // 设置应停止翻译服务
				rs.CloseTranslationService()                       // 关闭翻译服务
				log.Printf("[ROOM %s] 所有客户端断开，关闭翻译服务", rs.room.ID) // 记录日志
			}
		case message := <-rs.room.Broadcast: // 处理广播消息事件
			for client := range rs.room.Clients { // 遍历房间中的所有客户端
				select {
				case client.Send <- message: // 尝试发送消息到客户端
					// 成功发送，继续下一个客户端
				default: // 如果发送失败（通道阻塞）
					close(client.Send)              // 关闭客户端发送通道
					delete(rs.room.Clients, client) // 从房间中移除客户端
				}
			}
		}
	}
}

// CloseTranslationService 关闭翻译服务连接
func (rs *RoomService) CloseTranslationService() {
	// 更新监控中的连接状态
	monitor := GetTranslationMonitor()
	monitor.UpdateTranslationConnection(rs.room.ID, false, "", "")

	// 取消所有相关协程
	rs.cancel()

	rs.room.TranslationMux.Lock()         // 获取翻译连接互斥锁
	defer rs.room.TranslationMux.Unlock() // 函数结束时释放锁
	if rs.room.TranslationWS != nil {     // 如果翻译WebSocket连接存在
		_ = rs.room.TranslationWS.WriteMessage(websocket.BinaryMessage, []byte("END")) // 发送结束消息
		_ = rs.room.TranslationWS.Close()                                              // 关闭WebSocket连接
		rs.room.TranslationWS = nil                                                    // 清空连接对象
	}
	redisKey := fmt.Sprintf("room:%s:messages", rs.room.ID) // 构造Redis键名
	_ = RDB.Del(Ctx, redisKey).Err()                        // 删除Redis中的消息历史

	log.Printf("✅ [ROOM %s] 翻译服务已完全关闭，所有协程已停止", rs.room.ID)
}

// StartTranslationService 启动翻译服务连接
func (rs *RoomService) StartTranslationService() {
	rs.room.TranslationLock.Lock()         // 获取翻译服务锁
	defer rs.room.TranslationLock.Unlock() // 函数结束时释放锁

	rs.room.TranslationMux.Lock()     // 获取翻译连接互斥锁
	if rs.room.TranslationWS != nil { // 如果翻译连接已存在
		rs.room.TranslationMux.Unlock() // 释放锁
		return                          // 直接返回
	}
	rs.room.TranslationMux.Unlock() // 释放锁

	rs.room.ReconnectLock.Lock() // 获取重连锁
	if rs.room.IsReconnecting {  // 如果正在重连中
		rs.room.ReconnectLock.Unlock() // 释放锁
		return                         // 直接返回
	}
	rs.room.IsReconnecting = true  // 设置重连状态为true
	rs.room.ReconnectLock.Unlock() // 释放锁
	defer func() {                 // 设置延迟执行函数
		rs.room.ReconnectLock.Lock()   // 获取重连锁
		rs.room.IsReconnecting = false // 设置重连状态为false
		rs.room.ReconnectLock.Unlock() // 释放锁
	}()

	for { // 无限循环尝试连接
		select {
		case <-rs.ctx.Done(): // 检查上下文是否被取消
			log.Printf("🛑 [TRANSLATION %s] 翻译服务连接被取消", rs.room.ID)
			return
		default:
		}

		token, _ := GenerateJWT()                                                                           // 生成JWT令牌
		url := fmt.Sprintf("%s?token=%s&from_language=%s&to_language=%s&model=ultra&mute=False&multi=true", // 构造连接URL
			config.AppConfig.TranslationAPIURL, token, rs.room.FromLanguage, rs.room.ToLanguage)

		rootCAs, err := x509.SystemCertPool() // 获取系统证书池
		if err != nil {                       // 如果获取证书池失败
			log.Printf("[TRANSLATION %s] ❌ 加载系统证书池失败: %v", rs.room.ID, err) // 记录错误日志
			return                                                          // 退出函数
		}

		tlsConfig := &tls.Config{ // 创建TLS配置
			RootCAs: rootCAs, // 设置根证书
		}

		dialer := websocket.Dialer{ // 创建WebSocket拨号器
			TLSClientConfig: tlsConfig, // 设置TLS配置
		}

		conn, _, err := dialer.Dial(url, nil) // 尝试连接到翻译服务
		if err != nil {                       // 如果连接失败
			log.Printf("[TRANSLATION %s] ❌ 连接失败: %v", rs.room.ID, err) // 记录错误日志
			monitor := GetTranslationMonitor()
			monitor.RecordReconnect(rs.room.ID) // 记录重连尝试
			time.Sleep(2 * time.Second)         // 等待2秒后重试
			continue                            // 继续下一次循环
		}

		rs.room.TranslationMux.Lock()   // 获取翻译连接互斥锁
		rs.room.TranslationWS = conn    // 保存连接对象
		rs.room.TranslationMux.Unlock() // 释放锁

		// 更新监控中的连接状态
		monitor := GetTranslationMonitor()
		monitor.UpdateTranslationConnection(rs.room.ID, true, rs.room.FromLanguage, rs.room.ToLanguage)

		go rs.ReadFromTranslation() // 启动读取翻译消息的协程
		break                       // 退出循环
	}
}

// ReadFromTranslation 从翻译服务读取消息
func (rs *RoomService) ReadFromTranslation() {
	var currentBuffer strings.Builder // 创建字符串构建器用于累积消息
	var currentMessageID string       // 当前消息ID
	var lastProcessedPosition int     // 记录已处理的文本位置
	processor := audio.NewProcessor() // 创建音频处理器

	// 添加协程到监控
	monitor := GetTranslationMonitor()
	monitor.AddGoroutine(rs.room.ID, "translation_reader")
	defer monitor.RemoveGoroutine(rs.room.ID, "translation_reader")

	for { // 无限循环读取消息
		select {
		case <-rs.ctx.Done(): // 检查上下文是否被取消
			log.Printf("🛑 [TRANSLATION %s] 翻译消息读取被取消", rs.room.ID)
			return
		default:
		}

		rs.room.TranslationMux.Lock()   // 获取翻译连接互斥锁
		conn := rs.room.TranslationWS   // 获取当前连接
		rs.room.TranslationMux.Unlock() // 释放锁
		if conn == nil {                // 如果连接为空
			log.Printf("[TRANSLATION %s] ❌ nil WS，停止接收", rs.room.ID) // 记录错误日志
			return                                                   // 退出函数
		}

		msgType, message, err := conn.ReadMessage() // 读取消息
		if err != nil {                             // 如果读取失败
			log.Printf("[TRANSLATION %s] ❌ 读取失败: %v", rs.room.ID, err) // 记录错误日志

			// 检查是否是不支持的语言对错误 (close code 4001)
			if websocket.IsCloseError(err, 4001) {
				rs.SendUnsupportedLanguageMessage()
				log.Printf("[TRANSLATION %s] ❌ 不支持的语言对: from=%s, to=%s", rs.room.ID, rs.room.FromLanguage, rs.room.ToLanguage)
			}

			rs.room.TranslationMux.Lock()     // 获取翻译连接互斥锁
			if rs.room.TranslationWS != nil { // 如果连接存在
				_ = rs.room.TranslationWS.Close() // 关闭连接
				rs.room.TranslationWS = nil       // 清空连接对象
			}
			rs.room.TranslationMux.Unlock() // 释放锁

			go func() { // 启动协程处理重连
				select {
				case <-rs.ctx.Done(): // 检查上下文是否被取消
					log.Printf("🛑 [TRANSLATION %s] 重连协程被取消", rs.room.ID)
					return
				case <-time.After(2 * time.Second): // 等待2秒后重连
					if rs.room.ShouldStopTrans { // 如果应该停止翻译服务
						log.Printf("[TRANSLATION %s] ❎ 房间空，无需重连", rs.room.ID) // 记录日志
						return                                                // 退出协程
					}
					// 如果是不支持的语言对错误，不进行重连
					if websocket.IsCloseError(err, 4001) {
						log.Printf("[TRANSLATION %s] ❌ 语言对不支持，停止重连", rs.room.ID)
						return
					}
					rs.StartTranslationService() // 重新启动翻译服务
				}
			}()
			return // 退出函数
		}

		// 记录收到消息
		monitor.RecordMessage(rs.room.ID, msgType == websocket.BinaryMessage)

		switch msgType { // 根据消息类型处理
		case websocket.TextMessage: // 处理文本消息
			if len(message) == 0 || message[0] != '{' { // 如果消息为空或不是JSON格式
				log.Printf("[TRANSLATION %s] ⚠️ 非 JSON 消息: %s", rs.room.ID, string(message)) // 记录警告日志
				continue                                                                     // 跳过此消息
			}

			var data map[string]interface{}                        // 创建数据映射
			if err := json.Unmarshal(message, &data); err != nil { // 解析JSON消息
				log.Printf("[TRANSLATION %s] ❌ JSON 解析失败: %v", rs.room.ID, err) // 记录错误日志
				continue                                                        // 跳过此消息
			}

			text, _ := data["translation"].(string)         // 获取翻译文本
			partFinished, _ := data["part_finished"].(bool) // 获取部分完成状态
			lang, _ := data["language"].(string)            // 获取语言

			// 记录收到的翻译信息
			log.Printf("🌐 [单人模式 %s] 收到翻译消息: 语言=%s, 文本='%s', 完成状态=%v", rs.room.ID, lang, text, partFinished)

			currentBuffer.WriteString(text) // 将文本添加到缓冲区

			if currentMessageID == "" { // 如果当前消息ID为空
				currentMessageID = uuid.New().String() // 生成新的UUID作为消息ID
			}
			msgID := currentMessageID // 保存消息ID

			timestamp := ""   // 初始化时间戳
			if partFinished { // 如果部分完成
				timestamp = time.Now().Format(time.RFC3339) // 设置当前时间戳
			}

			final := map[string]interface{}{ // 创建最终消息映射
				"id":                   msgID,                  // 消息ID
				"translation":          currentBuffer.String(), // 翻译文本
				"language":             lang,                   // 语言
				"part_finished":        partFinished,           // 部分完成状态
				"timestamp":            timestamp,              // 时间戳
				"user":                 "",                     // 用户标识
				"reverseTranslation":   "",                     // 反向翻译文本（初始为空）
				"isReverseTranslation": false,                  // 是否为反向翻译
			}

			payload, err := json.Marshal(final) // 将映射转换为JSON
			if err != nil {                     // 如果转换失败
				log.Printf("[TRANSLATION %s] ❌ JSON 打包失败: %v", rs.room.ID, err) // 记录错误日志
				currentBuffer.Reset()                                           // 重置缓冲区
				currentMessageID = ""                                           // 清空消息ID
				return                                                          // 退出函数
			}

			rs.room.Broadcast <- payload // 广播消息到房间

			if partFinished { // 如果部分完成
				// partFinished=true：句子完成，查找是否已存在记录
				redisKey := fmt.Sprintf("room:%s:messages", rs.room.ID) // 构造Redis键名

				// 获取所有消息，查找当前msgID的记录
				messages, err := RDB.LRange(Ctx, redisKey, 0, -1).Result()
				if err != nil {
					log.Printf("[REDIS] ❌ 获取消息失败: %v", err)
					return
				}

				found := false
				for i, message := range messages {
					var existingMsg map[string]interface{}
					if err := json.Unmarshal([]byte(message), &existingMsg); err != nil {
						continue
					}

					if existingID, ok := existingMsg["id"].(string); ok && existingID == msgID {
						// 找到匹配的记录，更新为最终版本
						if err := RDB.LSet(Ctx, redisKey, int64(i), payload).Err(); err != nil {
							log.Printf("[REDIS] ❌ 更新失败: %v", err)
						} else {
							go rs.HandleReverseTranslation(msgID, lang)
						}
						found = true
						break
					}
				}

				if !found {
					// 如果没有找到对应记录，创建新记录
					if err := RDB.RPush(Ctx, redisKey, payload).Err(); err != nil {
						log.Printf("[REDIS] ❌ 存储失败: %v", err)
					} else {
						go rs.HandleReverseTranslation(msgID, lang)
					}
				}

				currentBuffer.Reset()     // 重置缓冲区
				currentMessageID = ""     // 清空消息ID
				lastProcessedPosition = 0 // 重置位置
			} else if endPos := rs.isSentenceEndFromPosition(currentBuffer.String(), lastProcessedPosition); endPos > lastProcessedPosition {
				// partFinished=false但句子完结：查找并更新当前messageID的记录
				lastProcessedPosition = endPos // 更新已处理位置
				redisKey := fmt.Sprintf("room:%s:messages", rs.room.ID)

				// 获取所有消息，查找当前msgID的记录
				messages, err := RDB.LRange(Ctx, redisKey, 0, -1).Result()
				if err != nil {
					log.Printf("[REDIS] ❌ 获取消息失败: %v", err)
					return
				}

				found := false
				for i, message := range messages {
					var existingMsg map[string]interface{}
					if err := json.Unmarshal([]byte(message), &existingMsg); err != nil {
						continue
					}

					if existingID, ok := existingMsg["id"].(string); ok && existingID == msgID {
						// 找到匹配的记录，更新它
						if err := RDB.LSet(Ctx, redisKey, int64(i), payload).Err(); err != nil {
							log.Printf("[REDIS] ❌ 更新失败: %v", err)
						} else {
							// 暂时注释掉句子完结时的回翻调用，仅在 part_finished 时触发
							// go rs.HandleReverseTranslation(msgID, lang)
						}
						found = true
						break
					}
				}

				if !found {
					// 如果没有找到对应记录，创建新记录
					if err := RDB.RPush(Ctx, redisKey, payload).Err(); err != nil {
						log.Printf("[REDIS] ❌ 存储失败: %v", err)
					} else {
						// 暂时注释掉句子完结时的回翻调用，仅在 part_finished 时触发
						// go rs.HandleReverseTranslation(msgID, lang)
					}
				}
				// 注意：这里不重置buffer和messageID，继续累积直到partFinished
			}

		case websocket.BinaryMessage: // 处理二进制消息（音频数据）
			if len(message) <= 20 { // 如果消息长度小于等于20字节
				log.Printf("[TRANSLATION %s] ⚠️ 音频太短跳过", rs.room.ID) // 记录警告日志
				continue                                             // 跳过此消息
			}
			trimmed := message[20:]                              // 去掉前20字节的头部
			resampled, err := processor.Resample(trimmed, false) // 重新采样音频
			if err != nil {                                      // 如果重采样失败
				log.Printf("[TRANSLATION %s] ❌ 重采样失败: %v", rs.room.ID, err) // 记录错误日志
				continue                                                    // 跳过此消息
			}
			rs.room.Broadcast <- resampled // 广播重采样后的音频数据
		}
	}
}

// HandleReverseTranslation 处理反向翻译逻辑
func (rs *RoomService) HandleReverseTranslation(messageID string, lang string) {
	// 检查Context是否被取消
	select {
	case <-rs.ctx.Done():
		log.Printf("🛑 [REVERSE %s] 反向翻译协程被取消", rs.room.ID)
		return
	default:
	}

	// 添加协程到监控
	monitor := GetTranslationMonitor()
	monitor.AddGoroutine(rs.room.ID, "reverse_translation")
	defer monitor.RemoveGoroutine(rs.room.ID, "reverse_translation")

	rs.room.TranslationQueueLock.Lock()                        // 获取翻译队列锁
	defer rs.room.TranslationQueueLock.Unlock()                // 函数结束时释放锁
	redisKey := fmt.Sprintf("room:%s:messages", rs.room.ID)    // 构造Redis键名
	messages, err := RDB.LRange(Ctx, redisKey, 0, -1).Result() // 获取所有历史消息
	if err != nil {                                            // 如果获取失败
		log.Printf("[REVERSE] ❌ 获取历史消息失败: %v", err) // 记录错误日志
		return                                      // 退出函数
	}

	var currentText string                 // 当前文本
	var user string                        // 用户标识
	var toLang string                      // 目标语言
	var targetIndex = -1                   // 目标索引
	var updatedItem map[string]interface{} // 更新的项目
	var contextPieces []string             // 上下文片段

	start := len(messages) - 6 // 开始索引（最近6条消息）
	if start < 0 {             // 如果开始索引小于0
		start = 0 // 设置为0
	}

	onlyOneMessage := len(messages[start:]) == 1 // 是否只有一条消息

	for i := start; i < len(messages); i++ { // 遍历消息
		var m map[string]interface{}                                    // 创建消息映射
		if err := json.Unmarshal([]byte(messages[i]), &m); err != nil { // 解析JSON消息
			continue // 解析失败跳过
		}

		idStr, _ := m["id"].(string)              // 获取消息ID
		u, _ := m["user"].(string)                // 获取用户标识
		t, _ := m["translation"].(string)         // 获取翻译文本
		rt, _ := m["reverseTranslation"].(string) // 获取反向翻译文本
		targetLang := m["language"].(string)      // 获取目标语言

		isCurrent := idStr == messageID // 是否为当前消息

		if isCurrent { // 如果是当前消息
			currentText = t // 设置当前文本

			// 一次性根据lang设定user和toLang
			if lang == rs.room.FromLanguage {
				user = "B:"
				toLang = rs.room.ToLanguage
			} else {
				user = "A:"
				toLang = rs.room.FromLanguage
			}

			// 直接更新消息体中的user字段
			m["user"] = user
			u = user
			targetIndex = i // 设置目标索引
			updatedItem = m // 设置更新项目（现在包含了更新后的user）
		}

		if !onlyOneMessage || !isCurrent { // 如果不是单一消息或不是当前消息
			if lang == targetLang { // 如果语言匹配目标语言
				contextPieces = append(contextPieces, u+t) // 添加用户和翻译文本到上下文
			} else {
				contextPieces = append(contextPieces, u+rt) // 添加用户和反向翻译文本到上下文
			}
		}
	}
	if currentText == "" || targetIndex == -1 { // 如果当前文本为空或目标索引为-1
		log.Printf("[REVERSE] ⚠️ 未找到匹配 ID=%s 的消息", messageID) // 记录警告日志
		return                                                // 退出函数
	}

	resultText := strings.Join(contextPieces, "\n") // 连接上下文片段

	const maxRetries = 3   // 最大重试次数
	var translated string  // 翻译结果
	var cost float64       // 翻译成本
	var translateErr error // 翻译错误

	for i := 0; i < maxRetries; i++ { // 重试循环
		timeout := 15 + float64(i*5) // 计算超时时间

		translated, cost, translateErr = translateconvtext.Translate( // 调用翻译服务
			toLang,      // 目标语言
			resultText,  // 上下文文本
			currentText, // 当前文本
			timeout,     // 超时时间
		)
		if translateErr == nil { // 如果翻译成功
			break // 退出循环
		}

		log.Printf("[REVERSE] ⚠️ 翻译尝试 %d 失败: %v", i+1, translateErr, cost) // 记录失败日志

		if strings.Contains(translateErr.Error(), "unexpected end of JSON input") { // 如果是JSON解析错误

			translated, cost, translateErr = translateconvtext.Translate( // 无上下文翻译
				toLang,      // 目标语言
				"",          // 空上下文
				currentText, // 当前文本
				20,          // 超时时间
			)
			if translateErr == nil { // 如果翻译成功
				break // 退出循环
			}
		}

		time.Sleep(time.Second * time.Duration(i+1)) // 等待后重试
	}

	if translateErr != nil { // 如果所有尝试都失败
		log.Printf("[REVERSE] ❌ 所有翻译尝试均失败: %v", translateErr) // 记录失败日志
		return                                                // 退出函数
	}

	updatedItem["reverseTranslation"] = translated // 设置反向翻译文本

	updatedPayload, err := json.Marshal(updatedItem) // 将更新项目转换为JSON
	if err != nil {                                  // 如果转换失败
		log.Printf("[REVERSE] ❌ 打包更新失败: %v", err) // 记录错误日志
		return                                    // 退出函数
	}

	if err := RDB.LSet(Ctx, redisKey, int64(targetIndex), updatedPayload).Err(); err != nil { // 更新Redis中的消息
		log.Printf("[REDIS] ❌ 更新失败: %v", err) // 记录错误日志
		return                                // 退出函数
	}

	rs.room.Broadcast <- updatedPayload // 广播更新后的消息
}

// isSentenceEndFromPosition 从指定位置开始检测文本是否包含完整句子，返回结束位置
func (rs *RoomService) isSentenceEndFromPosition(text string, startPos int) int {
	if len(text) == 0 || startPos >= len(text) {
		return -1
	}

	// 定义各种语言的句子结束符
	endMarkers := []string{
		"。", "！", "？", // 中文
		".", "!", "?", // 英文
		"؟", "۔", // 阿拉伯语
		"।", "॥", // 印地语
		"。", "！", "？", // 日文
		".", "!", "?", // 通用
	}

	// 只检测从startPos开始的部分
	searchText := text[startPos:]

	// 检查是否以句子结束符结尾
	trimmed := strings.TrimSpace(text)
	for _, marker := range endMarkers {
		if strings.HasSuffix(trimmed, marker) {
			endPos := len(trimmed)
			if endPos > startPos {
				return endPos
			}
		}
	}

	// 检查是否包含句子结束符后跟空格的模式（表示完整句子）
	for _, marker := range endMarkers {
		pattern := marker + " "
		if pos := strings.Index(searchText, pattern); pos != -1 {
			endPos := startPos + pos + len(marker)
			return endPos
		}
	}

	return -1
}

// isSentenceEnd 检测文本是否包含完整句子（保持向后兼容）
func (rs *RoomService) isSentenceEnd(text string) bool {
	return rs.isSentenceEndFromPosition(text, 0) > -1
}

// SendUnsupportedLanguageMessage 发送不支持的语言对消息给所有客户端
func (rs *RoomService) SendUnsupportedLanguageMessage() {
	unsupportedMessage := map[string]interface{}{
		"type":          "language_unsupported",
		"room_id":       rs.room.ID,
		"from_language": rs.room.FromLanguage,
		"to_language":   rs.room.ToLanguage,
		"message":       fmt.Sprintf("Sorry, translation from %s to %s is not currently supported", rs.room.FromLanguage, rs.room.ToLanguage),
		"status":        "unsupported",
	}

	messageBytes, err := json.Marshal(unsupportedMessage)
	if err != nil {
		log.Printf("[ROOM %s] ❌ 序列化不支持语言消息失败: %v", rs.room.ID, err)
		return
	}

	// 广播消息到房间
	rs.room.Broadcast <- messageBytes
}
