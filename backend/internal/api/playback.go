package api

import (
	"log/slog"
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

// validModes 合法的播放模式
var validModes = map[string]bool{
	"order":       true,
	"random":      true,
	"single-loop": true,
}

// isValidMode 校验播放模式
func isValidMode(mode string) bool {
	return validModes[mode]
}

// GetPlaybackState GET /api/playback/:identityId
func (h *Handler) GetPlaybackState(c *gin.Context) {
	identityID := c.Param("identityId")

	// 校验身份存在
	identity, err := h.identityRepo.GetByID(identityID)
	if err != nil {
		slog.Error("查询身份失败", "error", err, "id", identityID)
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
		slog.Error("查询播放状态失败", "error", err, "identity_id", identityID)
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

// GetPlaylistProgress GET /api/playlists/:id/progress
func (h *Handler) GetPlaylistProgress(c *gin.Context) {
	playlistID := c.Param("id")

	// 校验歌单存在
	playlist, err := h.playlistRepo.GetByID(playlistID)
	if err != nil {
		slog.Error("查询歌单失败", "error", err, "id", playlistID)
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

	state, err := h.playbackRepo.GetPlaylistState(playlistID)
	if err != nil {
		slog.Error("查询歌单记忆点失败", "error", err, "playlist_id", playlistID)
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    9001,
			Message: "内部错误",
		})
		return
	}
	if state == nil {
		// 没有记忆点时返回默认空状态
		c.JSON(http.StatusOK, model.APIResponse{
			Code:    0,
			Message: "ok",
			Data: model.PlaylistState{
				PlaylistID: playlistID,
				Position:   0,
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
		slog.Error("查询身份失败", "error", err, "id", identityID)
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

	// position 校验
	if req.Position != nil && *req.Position < 0 {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    9001,
			Message: "播放位置不能为负数",
		})
		return
	}

	// mode 校验（仅当传入时；未传入则保留数据库现有值）
	if req.Mode != nil && !isValidMode(*req.Mode) {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    9001,
			Message: "播放模式不合法",
		})
		return
	}

	// 校验传入的歌单存在
	if req.PlaylistID != nil && *req.PlaylistID != "" {
		playlist, err := h.playlistRepo.GetByID(*req.PlaylistID)
		if err != nil {
			slog.Error("校验歌单失败", "error", err, "playlist_id", *req.PlaylistID)
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

	// 校验传入的歌曲存在
	if req.SongID != nil && *req.SongID != "" {
		song, err := h.songRepo.GetByID(*req.SongID)
		if err != nil {
			slog.Error("校验歌曲失败", "error", err, "song_id", *req.SongID)
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

	// 校验歌曲属于歌单：计算合并后的生效 playlist_id/song_id
	// （仅用于校验；写入本身由 repo 层原子 upsert 完成，不依赖这里读到的值）
	effPlaylistID, effSongID := "", ""
	if req.PlaylistID == nil || req.SongID == nil {
		current, err := h.playbackRepo.GetByIdentity(identityID)
		if err != nil {
			slog.Error("查询当前播放状态失败", "error", err, "identity_id", identityID)
			c.JSON(http.StatusInternalServerError, model.APIResponse{
				Code:    9001,
				Message: "内部错误",
			})
			return
		}
		if current != nil {
			effPlaylistID = current.PlaylistID
			effSongID = current.SongID
		}
	}
	if req.PlaylistID != nil {
		effPlaylistID = *req.PlaylistID
	}
	if req.SongID != nil {
		effSongID = *req.SongID
	}
	if effPlaylistID != "" && effSongID != "" {
		inPlaylist, err := h.playlistRepo.ContainsSong(effPlaylistID, effSongID)
		if err != nil {
			slog.Error("校验歌曲与歌单关系失败", "error", err, "playlist_id", effPlaylistID, "song_id", effSongID)
			c.JSON(http.StatusInternalServerError, model.APIResponse{
				Code:    9001,
				Message: "内部错误",
			})
			return
		}
		if !inPlaylist {
			c.JSON(http.StatusBadRequest, model.APIResponse{
				Code:    ErrCodeSongNotInPlaylist,
				Message: "歌曲不在歌单中",
			})
			return
		}
	}

	// 原子合并保存播放状态 + 单曲进度 + 歌单记忆点（nil 字段保留现有值）
	state, err := h.playbackRepo.SaveWithProgress(identityID, req.PlaylistID, req.SongID, req.Position, req.Mode)
	if err != nil {
		slog.Error("保存播放状态失败", "error", err, "identity_id", identityID)
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    9001,
			Message: "内部错误",
		})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "ok",
		Data:    state,
	})
}
