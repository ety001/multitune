package api

import (
	"errors"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"

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
// 不再限定媒体根目录，始终返回根目录源，前端可从此进入任意目录。
func (h *Handler) ListStorageSources(c *gin.Context) {
	var sources []map[string]interface{}

	if h.cfg.LazyCatDeploy {
		sources = []map[string]interface{}{
			{
				"id":        "document",
				"name":      "文档",
				"path":      "/lzcapp/document",
				"available": true,
			},
			{
				"id":        "media",
				"name":      "媒体",
				"path":      "/lzcapp/media",
				"available": true,
			},
		}
	} else {
		sources = []map[string]interface{}{
			{
				"id":        "root",
				"name":      "根目录",
				"path":      "/",
				"available": true,
			},
		}
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
// path 为空时默认列出根目录；不再做 MEDIA_ROOT 沙箱校验。
func (h *Handler) ListDirectory(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		path = "/"
	}

	if h.cfg.LazyCatDeploy {
		if path == "/" {
			path = "/lzcapp"
		}
		if !strings.HasPrefix(path, "/lzcapp") {
			c.JSON(http.StatusBadRequest, model.APIResponse{
				Code:    ErrCodePathNotAccessible,
				Message: "无权限访问该路径",
			})
			return
		}
	}

	items, err := fsutil.ListDirectory(path)
	if err != nil {
		if errors.Is(err, fsutil.ErrPathNotFound) || errors.Is(err, fsutil.ErrNotADirectory) {
			c.JSON(http.StatusBadRequest, model.APIResponse{
				Code:    ErrCodePathNotAccessible,
				Message: err.Error(),
			})
			return
		}
		slog.Error("列出目录失败", "error", err, "path", path)
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    9001,
			Message: "内部错误",
		})
		return
	}

	parent := filepath.Dir(path)
	if parent == path || parent == "" {
		parent = path
	}
	if h.cfg.LazyCatDeploy && parent == "/" {
		parent = "/lzcapp"
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
