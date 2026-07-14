package api

import (
	"net/http"

	"github.com/ety001/multitune/internal/fsutil"
	"github.com/ety001/multitune/internal/model"
	"github.com/gin-gonic/gin"
)

// error codes for fs API
const (
	ErrCodeStorageSourceNotFound = 4001
	ErrCodePathNotAccessible     = 4002
)

// ListStorageSources GET /api/fs/sources
func (h *Handler) ListStorageSources(c *gin.Context) {
	sources, err := fsutil.ListSources(h.cfg.MediaRoot)
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
			Items: sources,
			Total: len(sources),
		},
	})
}

// ListDirectory GET /api/fs/list
func (h *Handler) ListDirectory(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		path = h.cfg.MediaRoot
	}

	if err := fsutil.ValidateMediaPath(h.cfg.MediaRoot, path); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    ErrCodePathNotAccessible,
			Message: err.Error(),
		})
		return
	}

	items, err := fsutil.ListDirectory(path)
	if err != nil {
		if err.Error() == "路径不存在" || err.Error() == "路径不是目录" {
			c.JSON(http.StatusBadRequest, model.APIResponse{
				Code:    ErrCodePathNotAccessible,
				Message: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    9001,
			Message: "内部错误",
		})
		return
	}

	parent := fsutil.ParentPath(h.cfg.MediaRoot, path)
	if parent == "" {
		parent = path
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "ok",
		Data: gin.H{
			"path":   path,
			"parent": parent,
			"items":  items,
		},
	})
}

// SearchSongs GET /api/fs/search
func (h *Handler) SearchSongs(c *gin.Context) {
	// 复用 /api/songs 的搜索能力
	h.ListSongs(c)
}
