package websocket

import (
	"bytes"
	"go-backEnd/internal/models"
	"go-backEnd/pkg/audio"
	"log"

	"github.com/gorilla/websocket"
)

// ReadPump 处理单终端模式下的客户端读取和转发，结束时会把音频缓冲写入磁盘
func ReadPump(c *models.Client, r *models.Room) {
	defer func() {
		if buf, ok := r.ClientAudioBuffers.Load(c); ok {
			audioBuffer := buf.(*bytes.Buffer)
			if audioBuffer.Len() > 0 {
				if err := audio.SavePCMFile(r.ID, audioBuffer.Bytes()); err != nil {
					log.Printf("[AUDIO] ❌ 保存PCM文件失败: %v", err)
				}
			}
		}

		r.Unregister <- c
		if err := c.Conn.Close(); err != nil {
			log.Printf("[CLIENT %s] ❌ 关闭连接失败: %v", r.ID, err)
		}
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			log.Printf("[CLIENT %s] ❌ 断开: %v", r.ID, err)
			break
		}
		if buf, ok := r.ClientAudioBuffers.Load(c); ok {
			buf.(*bytes.Buffer).Write(message)
		}
		r.TranslationMux.Lock()
		if r.TranslationWS != nil {
			_ = r.TranslationWS.WriteMessage(websocket.BinaryMessage, message)
		}
		r.TranslationMux.Unlock()
	}
}

// WritePump 将翻译结果写回单终端客户端
func WritePump(c *models.Client) {
	for message := range c.Send {
		_ = c.Conn.WriteMessage(websocket.BinaryMessage, message)
	}
}
