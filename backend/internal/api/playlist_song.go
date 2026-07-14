package api

import (
	"net/http"

	"github.com/ety001/multitune/internal/model"
	"github.com/gin-gonic/gin"
)

// addSongsRequest 添加歌曲请求
type addSongsRequest struct {
	SongIDs []string `json:"song_ids" binding:"required"`
}

// updateSongOrderRequest 调整顺序请求
type updateSongOrderRequest struct {
	SongIDs []string `json:"song_ids" binding:"required"`
}

// error codes for playlist song API
const (
	ErrCodeSongNotInPlaylist = 2004
	ErrCodePlaylistSongLimit = 2005
)

// AddSongsToPlaylist POST /api/playlists/:id/songs
func (h *Handler) AddSongsToPlaylist(c *gin.Context) {
	playlistID := c.Param("id")

	playlist, err := h.playlistRepo.GetByID(playlistID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    9001,
			Message: "内部错误",
		})
		return
	}
	if playlist == nil {
		c.JSON(http.StatusNotFound, model.APIResponse{
			Code:    ErrCodePlaylistNotFound,
			Message: "歌单不存在",
		})
		return
	}

	var req addSongsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    9001,
			Message: "请求参数错误",
		})
		return
	}

	// 校验歌曲存在
	for _, songID := range req.SongIDs {
		song, err := h.songRepo.GetByID(songID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.APIResponse{
				Code:    9001,
				Message: "内部错误",
			})
			return
		}
		if song == nil {
			c.JSON(http.StatusBadRequest, model.APIResponse{
				Code:    ErrCodeSongNotFound,
				Message: "歌曲不存在: " + songID,
			})
			return
		}
	}

	currentCount, err := h.playlistRepo.CountSongs(playlistID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    9001,
			Message: "内部错误",
		})
		return
	}
	if currentCount+len(req.SongIDs) > h.cfg.MaxSongsPerPlaylist {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    ErrCodePlaylistSongLimit,
			Message: "歌单歌曲数量超过上限",
		})
		return
	}

	added, err := h.playlistRepo.AddSongs(playlistID, req.SongIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    9001,
			Message: "内部错误",
		})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "ok",
		Data: gin.H{
			"added": added,
		},
	})
}

// RemoveSongFromPlaylist DELETE /api/playlists/:id/songs/:songId
func (h *Handler) RemoveSongFromPlaylist(c *gin.Context) {
	playlistID := c.Param("id")
	songID := c.Param("songId")

	playlist, err := h.playlistRepo.GetByID(playlistID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    9001,
			Message: "内部错误",
		})
		return
	}
	if playlist == nil {
		c.JSON(http.StatusNotFound, model.APIResponse{
			Code:    ErrCodePlaylistNotFound,
			Message: "歌单不存在",
		})
		return
	}

	if err := h.playlistRepo.RemoveSong(playlistID, songID); err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    9001,
			Message: "内部错误",
		})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "ok",
		Data:    nil,
	})
}

// UpdatePlaylistSongOrder PUT /api/playlists/:id/songs/order
func (h *Handler) UpdatePlaylistSongOrder(c *gin.Context) {
	playlistID := c.Param("id")

	playlist, err := h.playlistRepo.GetByID(playlistID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    9001,
			Message: "内部错误",
		})
		return
	}
	if playlist == nil {
		c.JSON(http.StatusNotFound, model.APIResponse{
			Code:    ErrCodePlaylistNotFound,
			Message: "歌单不存在",
		})
		return
	}

	var req updateSongOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    9001,
			Message: "请求参数错误",
		})
		return
	}

	if err := h.playlistRepo.UpdateSongOrder(playlistID, req.SongIDs); err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    9001,
			Message: "内部错误",
		})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "ok",
		Data:    nil,
	})
}
