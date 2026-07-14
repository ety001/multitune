package fsutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrPathNotFound 路径不存在
var ErrPathNotFound = errors.New("路径不存在")

// ErrNotADirectory 路径不是目录
var ErrNotADirectory = errors.New("路径不是目录")

// audioExtensions 支持的音频格式
var audioExtensions = map[string]bool{
	".mp3":  true,
	".flac": true,
	".m4a":  true,
	".aac":  true,
	".ogg":  true,
	".wav":  true,
}

// IsAudioFile 判断文件是否为支持的音频文件
func IsAudioFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return audioExtensions[ext]
}

// ValidateMediaPath 校验路径在 media root 下，防止目录遍历和软链接跳出
func ValidateMediaPath(mediaRoot, path string) error {
	if path == "" {
		return fmt.Errorf("路径不能为空")
	}

	// 清理路径
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("路径包含非法字符: %s", cleanPath)
	}

	// 解析 mediaRoot 的真实路径（解析软链接）
	realRoot, err := filepath.EvalSymlinks(mediaRoot)
	if err != nil {
		realRoot, err = filepath.Abs(mediaRoot)
		if err != nil {
			return fmt.Errorf("解析媒体根目录失败: %w", err)
		}
	}

	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("解析路径失败: %w", err)
	}

	// 逐级解析软链接：
	// 对路径的已存在部分做 EvalSymlinks，剩余部分拼接回去
	// 这样即使最终文件不存在，也能检测到中间目录的软链接
	realPath, err := resolveSymlinkPrefix(absPath)
	if err != nil {
		// 完全不存在时退回到绝对路径前缀检查
		realPath = absPath
	}

	if !strings.HasPrefix(realPath+string(filepath.Separator), realRoot+string(filepath.Separator)) && realPath != realRoot {
		return fmt.Errorf("路径不在允许的媒体目录内: %s", absPath)
	}

	return nil
}

// resolveSymlinkPrefix 解析路径中已存在部分的软链接
// 例如 /app/media/home/music/song.mp3 如果 music 是软链接，
// 解析 music 后再拼接 song.mp3
func resolveSymlinkPrefix(path string) (string, error) {
	// 先尝试对整个路径做 EvalSymlinks（路径完全存在的情况）
	real, err := filepath.EvalSymlinks(path)
	if err == nil {
		return real, nil
	}

	// 逐级回退，找到最长已存在前缀并解析
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	for {
		realDir, err := filepath.EvalSymlinks(dir)
		if err == nil {
			// 找到已存在的父目录，解析软链接后拼接剩余部分
			return filepath.Join(realDir, base), nil
		}
		if !os.IsNotExist(err) {
			// 其他错误（权限等），无法判断
			return path, err
		}
		// 父目录也不存在，继续回退
		parentDir := filepath.Dir(dir)
		base = filepath.Join(filepath.Base(dir), base)
		if parentDir == dir {
			// 已到根目录
			return path, nil
		}
		dir = parentDir
	}
}

// ListSources 列出媒体根目录下的存储源
func ListSources(mediaRoot string) ([]map[string]interface{}, error) {
	entries, err := os.ReadDir(mediaRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return []map[string]interface{}{}, nil
		}
		return nil, fmt.Errorf("读取媒体根目录失败: %w", err)
	}

	sources := []map[string]interface{}{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		id := entry.Name()
		path := filepath.Join(mediaRoot, id)
		available := isAccessible(path)
		sources = append(sources, map[string]interface{}{
			"id":        id,
			"name":      sourceDisplayName(id),
			"path":      path,
			"available": available,
		})
	}

	return sources, nil
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

// ParentPath 获取父目录路径
func ParentPath(mediaRoot, path string) string {
	absMediaRoot, _ := filepath.Abs(mediaRoot)
	absPath, _ := filepath.Abs(path)
	parent := filepath.Dir(absPath)
	if !strings.HasPrefix(parent, absMediaRoot) || parent == absPath {
		return ""
	}
	return parent
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

// sourceDisplayName 存储源显示名称
func sourceDisplayName(id string) string {
	switch id {
	case "home":
		return "主目录"
	case "usb":
		return "USB 存储"
	case "smb":
		return "SMB 共享"
	default:
		return id
	}
}
