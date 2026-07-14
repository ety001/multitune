package api

import (
	"net/http"

	"github.com/ety001/multitune/internal/model"
	"github.com/gin-gonic/gin"
)

// createPlaylistRequest 创建歌单请求
type createPlaylistRequest struct {
	Name      string `json:"name" binding:"required"`
	SortOrder int    `json:"sort_order"`
}

// updatePlaylistRequest 更新歌单请求
type updatePlaylistRequest struct {
	Name      *string `json:"name,omitempty"`
	CoverURL  *string `json:"cover_url,omitempty"`
	SortOrder *int    `json:"sort_order,omitempty"`
}

// error codes for playlist API
const (
	ErrCodePlaylistNotFound            = 2001
	ErrCodePlaylistNameEmpty           = 2002
	ErrCodePlaylistLimitExceeded       = 2003
	ErrCodeIdentityNotFoundForPlaylist = 1001 // 复用身份不存在错误码
)

// ListPlaylists GET /api/identities/:id/playlists
func (h *Handler) ListPlaylists(c *gin.Context) {
	identityID := c.Param("id")

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
			Code:    ErrCodeIdentityNotFoundForPlaylist,
			Message: "身份不存在",
		})
		return
	}

	playlists, err := h.playlistRepo.ListByIdentity(identityID)
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
		Data: model.ListResponse{
			Items: playlists,
			Total: len(playlists),
		},
	})
}

// CreatePlaylist POST /api/identities/:id/playlists
func (h *Handler) CreatePlaylist(c *gin.Context) {
	identityID := c.Param("id")

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
			Code:    ErrCodeIdentityNotFoundForPlaylist,
			Message: "身份不存在",
		})
		return
	}

	var req createPlaylistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    ErrCodePlaylistNameEmpty,
			Message: "歌单名称不能为空",
		})
		return
	}

	count, err := h.playlistRepo.CountByIdentity(identityID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    9001,
			Message: "内部错误",
		})
		return
	}
	if count >= h.cfg.MaxPlaylistsPerIdentity {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    ErrCodePlaylistLimitExceeded,
			Message: "歌单数量超过上限",
		})
		return
	}

	playlist, err := h.playlistRepo.Create(identityID, req.Name, req.SortOrder)
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
		Data:    playlist,
	})
}

// GetPlaylist GET /api/playlists/:id
func (h *Handler) GetPlaylist(c *gin.Context) {
	id := c.Param("id")
	playlist, err := h.playlistRepo.GetByID(id)
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

	// TODO: 实现歌曲列表查询后填充 songs
	resp := struct {
		model.Playlist
		Songs []model.Song `json:"songs"`
	}{
		Playlist: *playlist,
		Songs:    []model.Song{},
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "ok",
		Data:    resp,
	})
}

// UpdatePlaylist PUT /api/playlists/:id
func (h *Handler) UpdatePlaylist(c *gin.Context) {
	id := c.Param("id")
	var req updatePlaylistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    9001,
			Message: "请求参数错误",
		})
		return
	}

	if req.Name != nil && *req.Name == "" {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    ErrCodePlaylistNameEmpty,
			Message: "歌单名称不能为空",
		})
		return
	}

	playlist, err := h.playlistRepo.Update(id, req.Name, req.CoverURL, req.SortOrder)
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

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "ok",
		Data:    playlist,
	})
}

// DeletePlaylist DELETE /api/playlists/:id
func (h *Handler) DeletePlaylist(c *gin.Context) {
	id := c.Param("id")
	if err := h.playlistRepo.Delete(id); err != nil {
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
