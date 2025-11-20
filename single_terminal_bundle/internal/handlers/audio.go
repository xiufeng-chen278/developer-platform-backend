package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func ListAudio(w http.ResponseWriter, r *http.Request) {
	// 确保audio目录存在
	if err := os.MkdirAll("audio", 0755); err != nil {
		http.Error(w, "创建音频目录失败", http.StatusInternalServerError)
		return
	}

	files, err := os.ReadDir("audio")
	if err != nil {
		http.Error(w, "读取音频目录失败", http.StatusInternalServerError)
		return
	}

	var result []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".pcm") {
			result = append(result, "/audio/"+file.Name())
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func DeleteAudio(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("filename")
	if filename == "" || !strings.HasSuffix(filename, ".pcm") {
		http.Error(w, "缺少或非法文件名", http.StatusBadRequest)
		return
	}
	path := filepath.Join("audio", filename)
	if err := os.Remove(path); err != nil {
		http.Error(w, "删除失败", http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "删除成功: %s", filename)
}
