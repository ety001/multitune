package api

import (
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ety001/multitune/internal/fsutil"
	"github.com/ety001/multitune/internal/model"
	"github.com/gin-gonic/gin"
)

// StreamSong GET /api/songs/:id/stream
func (h *Handler) StreamSong(c *gin.Context) {
	id := c.Param("id")

	song, err := h.songRepo.GetByID(id)
	if err != nil {
		slog.Error("查询歌曲失败", "error", err, "song_id", id)
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    9001,
			Message: "内部错误",
		})
		return
	}
	if song == nil {
		c.JSON(http.StatusNotFound, model.APIResponse{
			Code:    ErrCodeSongNotFound,
			Message: "歌曲不存在",
		})
		return
	}

	file, err := os.Open(song.Path)
	if err != nil {
		slog.Error("打开歌曲文件失败", "error", err, "path", song.Path)
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, model.APIResponse{
				Code:    ErrCodeSongNotReadable,
				Message: "歌曲文件不存在",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    9001,
			Message: "内部错误",
		})
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		slog.Error("获取歌曲文件信息失败", "error", err, "path", song.Path)
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    9001,
			Message: "内部错误",
		})
		return
	}

	contentType := fsutil.ContentTypeByExt(filepath.Ext(song.Path))
	c.Header("Content-Type", contentType)
	c.Header("Accept-Ranges", "bytes")

	http.ServeContent(c.Writer, c.Request, filepath.Base(song.Path), stat.ModTime(), file)
}

// imageExtensions 同名专辑封面图片的查找优先级（小写，匹配时大小写不敏感）
var imageExtensions = []string{".jpg", ".jpeg", ".png", ".webp", ".gif"}

// findCoverPath 在歌曲同目录查找同名封面原图（小写扩展名优先，大小写不敏感兜底）
func findCoverPath(songPath string) string {
	dir := filepath.Dir(songPath)
	base := strings.TrimSuffix(filepath.Base(songPath), filepath.Ext(songPath))
	for _, ext := range imageExtensions {
		candidate := filepath.Join(dir, base+ext)
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate
		}
		// 大小写不敏感兜底（如 .JPG）：列目录里找同名不同扩展的匹配
		upper := filepath.Join(dir, base+strings.ToUpper(ext))
		if _, statErr := os.Stat(upper); statErr == nil {
			return upper
		}
	}
	return ""
}

// serveImageFile 以 http.ServeContent 输出图片文件
func serveImageFile(c *gin.Context, path, contentType, cacheControl string) {
	file, err := os.Open(path)
	if err != nil {
		slog.Error("打开图片失败", "error", err, "path", path)
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, model.APIResponse{
				Code:    ErrCodeSongCoverNotFound,
				Message: "图片不存在",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    9001,
			Message: "内部错误",
		})
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		slog.Error("获取图片信息失败", "error", err, "path", path)
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    9001,
			Message: "内部错误",
		})
		return
	}

	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", cacheControl)

	http.ServeContent(c.Writer, c.Request, filepath.Base(path), stat.ModTime(), file)
}

// CoverImage GET /api/songs/:id/cover[?size=thumb]
// 在歌曲文件同目录下查找同名图片并返回。size=thumb 返回 WebP 缩略图
// （懒生成、磁盘缓存于 CACHE_PATH，典型 10-30KB），生成失败时降级返回原图。
// 找不到封面返回 404，前端据 onerror 回退到默认封面。
func (h *Handler) CoverImage(c *gin.Context) {
	id := c.Param("id")

	song, err := h.songRepo.GetByID(id)
	if err != nil {
		slog.Error("查询歌曲失败", "error", err, "song_id", id)
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    9001,
			Message: "内部错误",
		})
		return
	}
	if song == nil {
		c.JSON(http.StatusNotFound, model.APIResponse{
			Code:    ErrCodeSongNotFound,
			Message: "歌曲不存在",
		})
		return
	}

	coverPath := findCoverPath(song.Path)
	if coverPath == "" {
		// 没有同名图片，返回 404，前端 onerror 回退默认封面
		c.JSON(http.StatusNotFound, model.APIResponse{
			Code:    ErrCodeSongCoverNotFound,
			Message: "无同名封面图片",
		})
		return
	}

	if c.Query("size") == "thumb" {
		thumb, err := h.ensureCoverThumb(song, coverPath)
		if err == nil {
			serveImageFile(c, thumb, "image/webp", "public, max-age=86400")
			return
		}
		// 缩略图生成失败（如 ffmpeg 缺失）时降级返回原图，不中断封面展示
		slog.Error("生成封面缩略图失败，降级返回原图", "error", err, "song_id", id)
	}

	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(coverPath)))
	if contentType == "" {
		contentType = "image/jpeg"
	}
	serveImageFile(c, coverPath, contentType, "public, max-age=3600")
}
