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

// CoverImage GET /api/songs/:id/cover
// 在歌曲文件同目录下查找同名图片（基础名 + 图片扩展名），找到则返回，
// 找不到返回 404，前端据 onerror 回退到默认封面。
// 例：/path/song.mp3 → 依次试 /path/song.jpg、.jpeg、.png、.webp、.gif
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

	dir := filepath.Dir(song.Path)
	base := strings.TrimSuffix(filepath.Base(song.Path), filepath.Ext(song.Path))

	// 按优先级试图片扩展名（目录大小写不敏感比对）
	var coverPath string
	for _, ext := range imageExtensions {
		candidate := filepath.Join(dir, base+ext)
		if _, statErr := os.Stat(candidate); statErr == nil {
			coverPath = candidate
			break
		}
		// 大小写不敏感兜底（如 .JPG）：列目录里找同名不同扩展的匹配
		upper := filepath.Join(dir, base+strings.ToUpper(ext))
		if _, statErr := os.Stat(upper); statErr == nil {
			coverPath = upper
			break
		}
	}

	if coverPath == "" {
		// 没有同名图片，返回 404，前端 onerror 回退默认封面
		c.JSON(http.StatusNotFound, model.APIResponse{
			Code:    ErrCodeSongCoverNotFound,
			Message: "无同名封面图片",
		})
		return
	}

	file, err := os.Open(coverPath)
	if err != nil {
		slog.Error("打开封面图片失败", "error", err, "path", coverPath)
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, model.APIResponse{
				Code:    ErrCodeSongCoverNotFound,
				Message: "封面图片不存在",
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
		slog.Error("获取封面图片信息失败", "error", err, "path", coverPath)
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    9001,
			Message: "内部错误",
		})
		return
	}

	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(coverPath)))
	if contentType == "" {
		contentType = "image/jpeg"
	}
	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "public, max-age=3600")

	http.ServeContent(c.Writer, c.Request, filepath.Base(coverPath), stat.ModTime(), file)
}
