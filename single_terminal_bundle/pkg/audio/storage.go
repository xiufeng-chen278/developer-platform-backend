package audio

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

func SavePCMFile(roomID string, audioData []byte) error {
	audioDir := "audio"
	if err := os.MkdirAll(audioDir, 0755); err != nil {
		return fmt.Errorf("创建音频目录失败: %w", err)
	}

	filename := filepath.Join(audioDir, fmt.Sprintf("%s.pcm", roomID))

	if err := ioutil.WriteFile(filename, audioData, 0644); err != nil {
		return fmt.Errorf("保存PCM文件失败: %w", err)
	}

	log.Printf("[AUDIO] ✅ 已保存会议 %s 的PCM音频文件: %s", roomID, filename)
	return nil
}

// SavePCMFileWithLanguage 保存带语言前缀的PCM文件
func SavePCMFileWithLanguage(roomID string, language string, audioData []byte) error {
	audioDir := "audio"
	if err := os.MkdirAll(audioDir, 0755); err != nil {
		return fmt.Errorf("创建音频目录失败: %w", err)
	}

	filename := filepath.Join(audioDir, fmt.Sprintf("%s_%s.pcm", language, roomID))

	if err := ioutil.WriteFile(filename, audioData, 0644); err != nil {
		return fmt.Errorf("保存PCM文件失败: %w", err)
	}

	log.Printf("[AUDIO] ✅ 已保存会议 %s 的%s语言PCM音频文件: %s", roomID, language, filename)
	return nil
}

// SavePCMFileWithHeader 保存带20字节头部的PCM文件
func SavePCMFileWithHeader(roomID string, language string, audioData []byte) error {
	audioDir := "audio"
	if err := os.MkdirAll(audioDir, 0755); err != nil {
		return fmt.Errorf("创建音频目录失败: %w", err)
	}

	filename := filepath.Join(audioDir, fmt.Sprintf("%s_%s_with_header.pcm", language, roomID))

	if err := ioutil.WriteFile(filename, audioData, 0644); err != nil {
		return fmt.Errorf("保存PCM文件失败: %w", err)
	}

	log.Printf("[AUDIO] ✅ 已保存会议 %s 的%s语言带头部PCM音频文件: %s", roomID, language, filename)
	return nil
}

// SavePCMFileProcessed 保存处理后的PCM文件（添加头部后再去除）
func SavePCMFileProcessed(roomID string, language string, audioData []byte) error {
	audioDir := "audio"
	if err := os.MkdirAll(audioDir, 0755); err != nil {
		return fmt.Errorf("创建音频目录失败: %w", err)
	}

	filename := filepath.Join(audioDir, fmt.Sprintf("%s_%s_processed.pcm", language, roomID))

	if err := ioutil.WriteFile(filename, audioData, 0644); err != nil {
		return fmt.Errorf("保存PCM文件失败: %w", err)
	}

	log.Printf("[AUDIO] ✅ 已保存会议 %s 的%s语言处理后PCM音频文件: %s", roomID, language, filename)
	return nil
}

// SaveMixedPCMFile 保存混合后的PCM音频文件
func SaveMixedPCMFile(roomID string, language string, audioData []byte) error {
	audioDir := "audio"
	if err := os.MkdirAll(audioDir, 0755); err != nil {
		return fmt.Errorf("创建音频目录失败: %w", err)
	}

	filename := filepath.Join(audioDir, fmt.Sprintf("mixed_%s_%s.pcm", language, roomID))

	if err := ioutil.WriteFile(filename, audioData, 0644); err != nil {
		return fmt.Errorf("保存混合PCM文件失败: %w", err)
	}

	log.Printf("[AUDIO] ✅ 已保存房间 %s 的%s语言混合PCM音频文件: %s, 大小: %d字节", roomID, language, filename, len(audioData))
	return nil
}

// SaveSentAudioFile 保存发送给翻译服务的音频文件（直接模式）
func SaveSentAudioFile(roomID string, language string, audioData []byte) error {
	audioDir := "audio"
	if err := os.MkdirAll(audioDir, 0755); err != nil {
		return fmt.Errorf("创建音频目录失败: %w", err)
	}

	// 保存带语言头部的完整音频
	filename := filepath.Join(audioDir, fmt.Sprintf("sent_%s_%s.pcm", language, roomID))
	if err := ioutil.WriteFile(filename, audioData, 0644); err != nil {
		return fmt.Errorf("保存发送音频文件失败: %w", err)
	}
	log.Printf("[AUDIO] ✅ 已保存房间 %s 的%s语言发送音频文件: %s, 大小: %d字节", roomID, language, filename, len(audioData))

	return nil
}

// SaveSentMixedAudioFile 保存发送给翻译服务的混合音频文件
func SaveSentMixedAudioFile(roomID string, language string, audioData []byte) error {
	audioDir := "audio"
	if err := os.MkdirAll(audioDir, 0755); err != nil {
		return fmt.Errorf("创建音频目录失败: %w", err)
	}

	// 保存带语言头部的完整混合音频
	filename := filepath.Join(audioDir, fmt.Sprintf("sent_mixed_%s_%s.pcm", language, roomID))
	if err := ioutil.WriteFile(filename, audioData, 0644); err != nil {
		return fmt.Errorf("保存发送混合音频文件失败: %w", err)
	}
	log.Printf("[AUDIO] ✅ 已保存房间 %s 的%s语言发送混合音频文件: %s, 大小: %d字节", roomID, language, filename, len(audioData))

	return nil
}

// SaveSentPureAudioFile 保存发送给翻译服务的纯音频文件（不含20字节头部）
func SaveSentPureAudioFile(roomID string, language string, audioData []byte) error {
	audioDir := "audio"
	if err := os.MkdirAll(audioDir, 0755); err != nil {
		return fmt.Errorf("创建音频目录失败: %w", err)
	}

	// 保存纯音频数据（不含语言头部）
	filename := filepath.Join(audioDir, fmt.Sprintf("sent_pure_%s_%s.pcm", language, roomID))
	if err := ioutil.WriteFile(filename, audioData, 0644); err != nil {
		return fmt.Errorf("保存发送纯音频文件失败: %w", err)
	}
	log.Printf("[AUDIO] ✅ 已保存房间 %s 的%s语言发送纯音频文件: %s, 大小: %d字节", roomID, language, filename, len(audioData))

	return nil
}

// SaveSentPureMixedAudioFile 保存发送给翻译服务的纯混合音频文件（不含20字节头部）
func SaveSentPureMixedAudioFile(roomID string, language string, audioData []byte) error {
	audioDir := "audio"
	if err := os.MkdirAll(audioDir, 0755); err != nil {
		return fmt.Errorf("创建音频目录失败: %w", err)
	}

	// 保存纯混合音频数据（不含语言头部）
	filename := filepath.Join(audioDir, fmt.Sprintf("sent_pure_mixed_%s_%s.pcm", language, roomID))
	if err := ioutil.WriteFile(filename, audioData, 0644); err != nil {
		return fmt.Errorf("保存发送纯混合音频文件失败: %w", err)
	}
	log.Printf("[AUDIO] ✅ 已保存房间 %s 的%s语言发送纯混合音频文件: %s, 大小: %d字节", roomID, language, filename, len(audioData))

	return nil
}

// SaveMixedPurePCMFile 保存混音后的纯音频文件（不含20字节头部）
func SaveMixedPurePCMFile(roomID string, language string, audioData []byte) error {
	audioDir := "audio"
	if err := os.MkdirAll(audioDir, 0755); err != nil {
		return fmt.Errorf("创建音频目录失败: %w", err)
	}

	filename := filepath.Join(audioDir, fmt.Sprintf("mixed_pure_%s_%s.pcm", language, roomID))
	if err := ioutil.WriteFile(filename, audioData, 0644); err != nil {
		return fmt.Errorf("保存混音纯音频文件失败: %w", err)
	}
	log.Printf("[AUDIO] ✅ 已保存房间 %s 的%s语言混音纯音频文件: %s, 大小: %d字节", roomID, language, filename, len(audioData))

	return nil
}

// SaveMixedSentPCMFile 保存混音后发送给翻译服务的音频文件（含20字节头部）
func SaveMixedSentPCMFile(roomID string, language string, audioData []byte) error {
	audioDir := "audio"
	if err := os.MkdirAll(audioDir, 0755); err != nil {
		return fmt.Errorf("创建音频目录失败: %w", err)
	}

	filename := filepath.Join(audioDir, fmt.Sprintf("mixed_sent_%s_%s.pcm", language, roomID))
	if err := ioutil.WriteFile(filename, audioData, 0644); err != nil {
		return fmt.Errorf("保存混音发送音频文件失败: %w", err)
	}
	log.Printf("[AUDIO] ✅ 已保存房间 %s 的%s语言混音发送音频文件: %s, 大小: %d字节", roomID, language, filename, len(audioData))

	return nil
}
