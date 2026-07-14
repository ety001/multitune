package fsutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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

// ValidateMediaPath 校验路径在 media root 下且不存在目录遍历
func ValidateMediaPath(mediaRoot, path string) error {
	if path == "" {
		return fmt.Errorf("路径不能为空")
	}

	// 清理路径
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("路径包含非法字符: %s", cleanPath)
	}

	// 解析绝对路径
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("解析路径失败: %w", err)
	}

	absMediaRoot, err := filepath.Abs(mediaRoot)
	if err != nil {
		return fmt.Errorf("解析媒体根目录失败: %w", err)
	}

	// 确保路径在 mediaRoot 下
	if !strings.HasPrefix(absPath, absMediaRoot) {
		return fmt.Errorf("路径不在允许的媒体目录内: %s", absPath)
	}

	// 检查软链接是否跳出 mediaRoot
	info, err := os.Lstat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 路径不存在时只完成遍历校验，调用方再处理
		}
		return fmt.Errorf("获取路径信息失败: %w", err)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(absPath)
		if err != nil {
			return fmt.Errorf("读取软链接失败: %w", err)
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(absPath), target)
		}
		absTarget, err := filepath.Abs(target)
		if err != nil {
			return fmt.Errorf("解析软链接目标失败: %w", err)
		}
		if !strings.HasPrefix(absTarget, absMediaRoot) {
			return fmt.Errorf("软链接目标不在允许的媒体目录内")
		}
	}

	return nil
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

// ListDirectory 列出目录内容
func ListDirectory(path string) ([]map[string]interface{}, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("路径不存在")
		}
		return nil, fmt.Errorf("获取目录信息失败: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("路径不是目录")
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("读取目录失败: %w", err)
	}

	items := []map[string]interface{}{}
	for _, entry := range entries {
		item := map[string]interface{}{
			"name": entry.Name(),
			"path": filepath.Join(path, entry.Name()),
		}
		if entry.IsDir() {
			item["type"] = "dir"
		} else {
			item["type"] = "file"
			item["is_audio"] = IsAudioFile(entry.Name())
			if info, err := entry.Info(); err == nil {
				item["size"] = info.Size()
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
