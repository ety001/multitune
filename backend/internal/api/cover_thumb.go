package api

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ety001/multitune/internal/model"
)

// coverThumbSize 缩略图最长边像素
const coverThumbSize = 300

// coverThumbPrefix 缩略图缓存文件名前缀，用于清理同歌曲旧版本
func coverThumbPrefix(songID string) string {
	return songID + "-"
}

// thumbCachePath 计算封面缩略图缓存路径。
// 文件名包含原图 size+mtime：原图变更后 key 变化，旧缓存自然失效。
func (h *Handler) thumbCachePath(song *model.Song, coverPath string) (string, error) {
	info, err := os.Stat(coverPath)
	if err != nil {
		return "", fmt.Errorf("获取封面原图信息失败: %w", err)
	}
	name := fmt.Sprintf("%s-%dx%d.webp", song.ID, info.Size(), info.ModTime().Unix())
	return filepath.Join(h.cfg.CachePath, name), nil
}

// ensureCoverThumb 返回缩略图路径，缓存未命中时用 ffmpeg 生成 WebP。
// 生成的临时文件先写后 rename，多请求并发时最多重复生成一次，结果一致。
func (h *Handler) ensureCoverThumb(song *model.Song, coverPath string) (string, error) {
	cacheFile, err := h.thumbCachePath(song, coverPath)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(cacheFile); err == nil {
		return cacheFile, nil
	}

	tmp, err := os.CreateTemp(h.cfg.CachePath, "thumb-*.tmp")
	if err != nil {
		return "", fmt.Errorf("创建临时文件失败: %w", err)
	}
	tmpName := tmp.Name()
	tmp.Close()

	// -f webp：临时文件无扩展名，显式指定输出格式；-frames:v 1：动图封面只取首帧
	cmd := exec.Command("ffmpeg", "-y", "-v", "error",
		"-i", coverPath,
		"-vf", fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease", coverThumbSize, coverThumbSize),
		"-frames:v", "1",
		"-quality", "75",
		"-f", "webp",
		tmpName)
	if out, err := cmd.CombinedOutput(); err != nil {
		os.Remove(tmpName)
		return "", fmt.Errorf("ffmpeg 生成缩略图失败: %w, output: %s", err, strings.TrimSpace(string(out)))
	}
	if err := os.Rename(tmpName, cacheFile); err != nil {
		os.Remove(tmpName)
		return "", fmt.Errorf("写入缓存失败: %w", err)
	}

	h.cleanStaleThumbs(song.ID, cacheFile)
	return cacheFile, nil
}

// cleanStaleThumbs 删除同一歌曲旧版本的缩略图（原图变更后 key 变化遗留的文件）
func (h *Handler) cleanStaleThumbs(songID, keepFile string) {
	entries, err := os.ReadDir(h.cfg.CachePath)
	if err != nil {
		return
	}
	prefix := coverThumbPrefix(songID)
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, prefix) && name != filepath.Base(keepFile) {
			if err := os.Remove(filepath.Join(h.cfg.CachePath, name)); err != nil {
				slog.Warn("清理过期缩略图失败", "error", err, "file", name)
			}
		}
	}
}
