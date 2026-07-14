package api

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

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

	if err := fsutil.ValidateMediaPath(h.cfg.MediaRoot, song.Path); err != nil {
		slog.Error("歌曲路径校验失败", "error", err, "path", song.Path)
		c.JSON(http.StatusForbidden, model.APIResponse{
			Code:    ErrCodeSongNotReadable,
			Message: "歌曲文件不可访问",
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
