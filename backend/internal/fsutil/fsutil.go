package fsutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrPathNotFound 路径不存在
var ErrPathNotFound = fmt.Errorf("路径不存在")

// ErrNotADirectory 路径不是目录
var ErrNotADirectory = fmt.Errorf("路径不是目录")

// audioExtensions 支持的音频格式
var audioExtensions = map[string]bool{
	".mp3":  true,
	".flac": true,
	".m4a":  true,
	".aac":  true,
	".ogg":  true,
	".wav":  true,
}

// contentTypes 音频文件扩展名到 MIME 类型映射
var contentTypes = map[string]string{
	".mp3":  "audio/mpeg",
	".flac": "audio/flac",
	".m4a":  "audio/mp4",
	".aac":  "audio/aac",
	".ogg":  "audio/ogg",
	".wav":  "audio/wav",
}

// IsAudioFile 判断文件是否为支持的音频文件
func IsAudioFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return audioExtensions[ext]
}

// ContentTypeByExt 根据扩展名返回音频 MIME 类型
func ContentTypeByExt(ext string) string {
	ct := contentTypes[strings.ToLower(ext)]
	if ct == "" {
		return "application/octet-stream"
	}
	return ct
}

// DirEntry 目录条项
type DirEntry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Type    string `json:"type"`
	IsAudio bool   `json:"is_audio,omitempty"`
	Size    int64  `json:"size,omitempty"`
}

// ListDirectory 列出目录内容
func ListDirectory(path string) ([]DirEntry, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrPathNotFound
		}
		return nil, fmt.Errorf("获取目录信息失败: %w", err)
	}
	if !info.IsDir() {
		return nil, ErrNotADirectory
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("读取目录失败: %w", err)
	}

	items := []DirEntry{}
	for _, entry := range entries {
		item := DirEntry{
			Name: entry.Name(),
			Path: filepath.Join(path, entry.Name()),
		}
		if entry.IsDir() {
			item.Type = "dir"
		} else {
			item.Type = "file"
			item.IsAudio = IsAudioFile(entry.Name())
			if info, err := entry.Info(); err == nil {
				item.Size = info.Size()
			}
		}
		items = append(items, item)
	}

	return items, nil
}

// isAccessible 检查路径是否可读
func isAccessible(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if !info.IsDir() {
		return false
	}
	// 尝试读取目录内容验证可访问性
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	_, err = f.Readdirnames(1)
	return err == nil || err.Error() == "EOF"
}
