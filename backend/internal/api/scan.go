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
	ErrCodeScanBusy          = 4003
	ErrCodeSongNotFound      = 3001
	ErrCodeSongNotReadable   = 3002
	ErrCodeSongCoverNotFound = 3003
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

// createScanJobRequest 创建扫描任务请求
type createScanJobRequest struct {
	Paths      []string `json:"paths" binding:"required"`
	PlaylistID string   `json:"playlist_id" binding:"required"`
}

// CreateScanJob POST /api/scan/jobs
func (h *Handler) CreateScanJob(c *gin.Context) {
	var req createScanJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    4001,
			Message: "请求参数错误：paths 和 playlist_id 不能为空",
		})
		return
	}

	if len(req.Paths) == 0 {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    4001,
			Message: "至少选择一个路径",
		})
		return
	}

	// 校验歌单存在
	playlist, err := h.playlistRepo.GetByID(req.PlaylistID)
	if err != nil {
		slog.Error("查询歌单失败", "error", err)
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    9001,
			Message: "内部错误",
		})
		return
	}
	if playlist == nil {
		c.JSON(http.StatusNotFound, model.APIResponse{
			Code:    2001,
			Message: "歌单不存在",
		})
		return
	}

	job, err := h.scanJobRepo.Create(req.PlaylistID)
	if err != nil {
		slog.Error("创建扫描任务失败", "error", err)
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    9001,
			Message: "内部错误",
		})
		return
	}

	// 异步执行扫描
	go h.runScanJob(job, req.Paths)

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "ok",
		Data:    job,
	})
}

// GetScanJob GET /api/scan/jobs/:id
func (h *Handler) GetScanJob(c *gin.Context) {
	id := c.Param("id")
	job, err := h.scanJobRepo.GetByID(id)
	if err != nil {
		slog.Error("查询扫描任务失败", "error", err, "id", id)
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    9001,
			Message: "内部错误",
		})
		return
	}
	if job == nil {
		c.JSON(http.StatusNotFound, model.APIResponse{
			Code:    4004,
			Message: "扫描任务不存在",
		})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "ok",
		Data:    job,
	})
}

// runScanJob 后台执行扫描任务并更新进度
func (h *Handler) runScanJob(job *model.ScanJob, paths []string) {
	update := func() {
		if err := h.scanJobRepo.Update(job); err != nil {
			slog.Error("更新扫描任务进度失败", "error", err, "job_id", job.ID)
		}
	}

	job.Status = "counting"
	update()

	result, err := h.scanner.ScanPaths(paths, func(current, total int) {
		if total == 0 {
			return
		}
		if job.Status != "scanning" {
			job.Status = "scanning"
		}
		job.Total = total
		job.Current = current
		update()
	})

	if err != nil {
		if errors.Is(err, scanner.ErrScanInProgress) {
			job.Status = "error"
			job.Message = "扫描任务正在进行中"
		} else {
			job.Status = "error"
			job.Message = err.Error()
		}
		update()
		return
	}

	// 扫描完成，将歌曲添加到歌单
	songIDs := make([]string, 0, len(result.Songs))
	for _, song := range result.Songs {
		songIDs = append(songIDs, song.ID)
	}

	if len(songIDs) > 0 {
		added, err := h.playlistRepo.AddSongs(job.PlaylistID, songIDs)
		if err != nil {
			job.Status = "error"
			job.Message = "添加歌曲到歌单失败: " + err.Error()
			update()
			return
		}
		job.Added = added
	}

	job.Status = "done"
	job.Current = job.Total
	job.Updated = result.Updated
	update()
}

// batchSongsRequest 批量查询歌曲详情请求
type batchSongsRequest struct {
	IDs []string `json:"ids"`
}

// batchSongsResponse 批量查询歌曲详情响应。
// 注意：返回的歌曲顺序不保证与请求 ids 顺序一致（SQL IN 不保序），
// 调用方需按自己持有的 id 顺序自行重排。
type batchSongsResponse struct {
	Songs []model.Song `json:"songs"`
}

// BatchGetSongs POST /api/songs/batch
// 按歌曲 ID 集合批量查询详情，供车机版虚拟列表按需加载。
// 单次最多 100 个 id，防止滥用。
func (h *Handler) BatchGetSongs(c *gin.Context) {
	var req batchSongsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	if len(req.IDs) == 0 {
		c.JSON(http.StatusOK, model.APIResponse{
			Code:    0,
			Message: "ok",
			Data: batchSongsResponse{
				Songs: make([]model.Song, 0),
			},
		})
		return
	}

	if len(req.IDs) > 100 {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    400,
			Message: "单次最多查询 100 首歌曲",
		})
		return
	}

	songs, err := h.songRepo.ListByIDs(req.IDs)
	if err != nil {
		slog.Error("批量查询歌曲失败", "error", err, "count", len(req.IDs))
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    9001,
			Message: "内部错误",
		})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "ok",
		Data: batchSongsResponse{
			Songs: songs,
		},
	})
}
