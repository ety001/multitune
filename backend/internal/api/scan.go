package api

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/ety001/multitune/internal/model"
	"github.com/ety001/multitune/internal/scanner"
	"github.com/gin-gonic/gin"
)

// scanRequest 扫描请求
type scanRequest struct {
	Path string `json:"path" binding:"required"`
}

// error codes for scan API
const (
	ErrCodeScanBusy        = 4003
	ErrCodeSongNotFound    = 3001
	ErrCodeSongNotReadable = 3002
)

// ScanSongs POST /api/scan
func (h *Handler) ScanSongs(c *gin.Context) {
	var req scanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    ErrCodePathNotAccessible,
			Message: "路径不能为空",
		})
		return
	}

	result, err := h.scanner.ScanPath(req.Path)
	if err != nil {
		if errors.Is(err, scanner.ErrScanInProgress) {
			c.JSON(http.StatusConflict, model.APIResponse{
				Code:    ErrCodeScanBusy,
				Message: "扫描任务正在进行中",
			})
			return
		}
		slog.Error("扫描失败", "error", err, "path", req.Path)
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    9001,
			Message: "内部错误",
		})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "ok",
		Data:    result,
	})
}

// ListSongs GET /api/songs
func (h *Handler) ListSongs(c *gin.Context) {
	query := c.Query("q")
	source := c.Query("source")
	limit := parseInt(c.DefaultQuery("limit", "20"), 20)
	offset := parseInt(c.DefaultQuery("offset", "0"), 0)

	if limit > 200 {
		limit = 200
	}
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	songs, total, err := h.songRepo.List(query, source, limit, offset)
	if err != nil {
		slog.Error("查询歌曲列表失败", "error", err)
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    9001,
			Message: "内部错误",
		})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "ok",
		Data: model.ListResponse{
			Items: songs,
			Total: total,
		},
	})
}

// GetSong GET /api/songs/:id
func (h *Handler) GetSong(c *gin.Context) {
	id := c.Param("id")
	song, err := h.songRepo.GetByID(id)
	if err != nil {
		slog.Error("查询歌曲失败", "error", err, "id", id)
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

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "ok",
		Data:    song,
	})
}

// parseInt 解析整数，失败返回默认值
func parseInt(s string, defaultValue int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return defaultValue
	}
	return n
}
