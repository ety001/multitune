package api

import (
	"net/http"

	"github.com/ety001/multitune/internal/model"
	"github.com/gin-gonic/gin"
)

// savePlaybackRequest 保存播放状态请求
type savePlaybackRequest struct {
	PlaylistID *string `json:"playlist_id,omitempty"`
	SongID     *string `json:"song_id,omitempty"`
	Position   *int    `json:"position,omitempty"`
	Mode       *string `json:"mode,omitempty"`
}

// error codes for playback API
const (
	ErrCodePlaybackNotFound = 5001
)

// GetPlaybackState GET /api/playback/:identityId
func (h *Handler) GetPlaybackState(c *gin.Context) {
	identityID := c.Param("identityId")

	// 校验身份存在
	identity, err := h.identityRepo.GetByID(identityID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    9001,
			Message: "内部错误",
		})
		return
	}
	if identity == nil {
		c.JSON(http.StatusNotFound, model.APIResponse{
			Code:    ErrCodeIdentityNotFound,
			Message: "身份不存在",
		})
		return
	}

	state, err := h.playbackRepo.GetByIdentity(identityID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    9001,
			Message: "内部错误",
		})
		return
	}
	if state == nil {
		// 没有状态时返回默认空状态
		c.JSON(http.StatusOK, model.APIResponse{
			Code:    0,
			Message: "ok",
			Data: model.PlaybackState{
				IdentityID: identityID,
				Position:   0,
				Mode:       "order",
			},
		})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "ok",
		Data:    state,
	})
}

// SavePlaybackState POST /api/playback/:identityId
func (h *Handler) SavePlaybackState(c *gin.Context) {
	identityID := c.Param("identityId")

	// 校验身份存在
	identity, err := h.identityRepo.GetByID(identityID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    9001,
			Message: "内部错误",
		})
		return
	}
	if identity == nil {
		c.JSON(http.StatusNotFound, model.APIResponse{
			Code:    ErrCodeIdentityNotFound,
			Message: "身份不存在",
		})
		return
	}

	var req savePlaybackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    9001,
			Message: "请求参数错误",
		})
		return
	}

	// 获取当前状态作为默认值
	current, err := h.playbackRepo.GetByIdentity(identityID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    9001,
			Message: "内部错误",
		})
		return
	}
	if current == nil {
		current = &model.PlaybackState{
			IdentityID: identityID,
			Position:   0,
			Mode:       "order",
		}
	}

	playlistID := current.PlaylistID
	songID := current.SongID
	position := current.Position
	mode := current.Mode

	if req.PlaylistID != nil {
		playlistID = *req.PlaylistID
		if playlistID != "" {
			playlist, err := h.playlistRepo.GetByID(playlistID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.APIResponse{
					Code:    9001,
					Message: "内部错误",
				})
				return
			}
			if playlist == nil {
				c.JSON(http.StatusBadRequest, model.APIResponse{
					Code:    ErrCodePlaylistNotFound,
					Message: "歌单不存在",
				})
				return
			}
		}
	}
	if req.SongID != nil {
		songID = *req.SongID
		if songID != "" {
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
					Message: "歌曲不存在",
				})
				return
			}
		}
	}
	if req.Position != nil {
		position = *req.Position
	}
	if req.Mode != nil {
		mode = *req.Mode
	}

	// 校验播放模式
	if mode != "order" && mode != "random" && mode != "single-loop" {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    9001,
			Message: "播放模式不合法",
		})
		return
	}

	state, err := h.playbackRepo.Save(identityID, playlistID, songID, position, mode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    9001,
			Message: "内部错误",
		})
		return
	}

	// 同时保存单曲进度
	if songID != "" {
		if err := h.playbackRepo.SaveSongProgress(identityID, songID, position); err != nil {
			c.JSON(http.StatusInternalServerError, model.APIResponse{
				Code:    9001,
				Message: "内部错误",
			})
			return
		}
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "ok",
		Data:    state,
	})
}
