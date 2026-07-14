package api

import (
	"net/http"

	"github.com/ety001/multitune/internal/model"
	"github.com/gin-gonic/gin"
)

// createIdentityRequest 创建身份请求
type createIdentityRequest struct {
	Name        string `json:"name" binding:"required"`
	AvatarColor string `json:"avatar_color" binding:"required"`
	SortOrder   int    `json:"sort_order"`
}

// updateIdentityRequest 更新身份请求
type updateIdentityRequest struct {
	Name        *string `json:"name,omitempty"`
	AvatarColor *string `json:"avatar_color,omitempty"`
	SortOrder   *int    `json:"sort_order,omitempty"`
}

// identityDetailResponse 身份详情响应
type identityDetailResponse struct {
	model.Identity
	Playlists []identityPlaylist `json:"playlists"`
}

type identityPlaylist struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	SongCount int    `json:"song_count"`
}

// error codes for identity API
const (
	ErrCodeIdentityNotFound      = 1001
	ErrCodeIdentityNameEmpty     = 1002
	ErrCodeIdentityLimitExceeded = 1003
)

// ListIdentities GET /api/identities
func (h *Handler) ListIdentities(c *gin.Context) {
	identities, err := h.identityRepo.List()
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
			Items: identities,
			Total: len(identities),
		},
	})
}

// CreateIdentity POST /api/identities
func (h *Handler) CreateIdentity(c *gin.Context) {
	var req createIdentityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    ErrCodeIdentityNameEmpty,
			Message: "身份名称和颜色不能为空",
		})
		return
	}

	count, err := h.identityRepo.Count()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    9001,
			Message: "内部错误",
		})
		return
	}
	if count >= h.cfg.MaxIdentities {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    ErrCodeIdentityLimitExceeded,
			Message: "身份数量超过上限",
		})
		return
	}

	identity, err := h.identityRepo.Create(req.Name, req.AvatarColor, req.SortOrder)
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
		Data:    identity,
	})
}

// GetIdentity GET /api/identities/:id
func (h *Handler) GetIdentity(c *gin.Context) {
	id := c.Param("id")
	identity, err := h.identityRepo.GetByID(id)
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

	// TODO: 实现歌单列表查询后填充 playlists
	resp := identityDetailResponse{
		Identity:  *identity,
		Playlists: []identityPlaylist{},
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "ok",
		Data:    resp,
	})
}

// UpdateIdentity PUT /api/identities/:id
func (h *Handler) UpdateIdentity(c *gin.Context) {
	id := c.Param("id")
	var req updateIdentityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    9001,
			Message: "请求参数错误",
		})
		return
	}

	if req.Name != nil && *req.Name == "" {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    ErrCodeIdentityNameEmpty,
			Message: "身份名称不能为空",
		})
		return
	}

	identity, err := h.identityRepo.Update(id, req.Name, req.AvatarColor, req.SortOrder)
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

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "ok",
		Data:    identity,
	})
}

// DeleteIdentity DELETE /api/identities/:id
func (h *Handler) DeleteIdentity(c *gin.Context) {
	id := c.Param("id")
	if err := h.identityRepo.Delete(id); err != nil {
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

// SetDefaultIdentity POST /api/identities/:id/default
func (h *Handler) SetDefaultIdentity(c *gin.Context) {
	id := c.Param("id")
	identity, err := h.identityRepo.SetDefault(id)
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

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "ok",
		Data: gin.H{
			"id":         identity.ID,
			"is_default": identity.IsDefault,
		},
	})
}
