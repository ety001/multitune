package api

import (
	"errors"
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/ety001/multitune/internal/fsutil"
	"github.com/ety001/multitune/internal/model"
	"github.com/gin-gonic/gin"
)

// error codes for fs API
const (
	ErrCodeStorageSourceNotFound = 4001
	ErrCodePathNotAccessible     = 4002
	ErrCodeUserNotIdentified     = 4005
)

// ListStorageSources GET /api/fs/sources
// 懒猫部署时按当前登录用户返回可用存储源：媒体目录（全体共享）与
// 该用户自己的文档目录（/lzcapp/document/<username>）。
func (h *Handler) ListStorageSources(c *gin.Context) {
	var sources []map[string]interface{}

	if h.cfg.LazyCatDeploy {
		if _, err := h.lazycatAllowedRoots(c); err != nil {
			c.JSON(http.StatusUnauthorized, model.APIResponse{
				Code:    ErrCodeUserNotIdentified,
				Message: "无法识别当前用户，拒绝访问文件系统",
			})
			return
		}

		docDir := filepath.Join(h.cfg.LazyCatDocumentRoot, h.lazycatUsername(c))
		sources = []map[string]interface{}{
			{
				"id":        "media",
				"name":      "媒体",
				"path":      h.cfg.LazyCatMediaRoot,
				"available": fsutil.IsDirReadable(h.cfg.LazyCatMediaRoot),
			},
			{
				"id":        "document",
				"name":      "我的文档",
				"path":      docDir,
				"available": fsutil.IsDirReadable(docDir),
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
// path 为空时默认列出根目录。懒猫部署时仅允许访问当前用户的文档目录与媒体目录。
func (h *Handler) ListDirectory(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		path = "/"
	}

	var roots []string
	if h.cfg.LazyCatDeploy {
		allowed, err := h.lazycatAllowedRoots(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, model.APIResponse{
				Code:    ErrCodeUserNotIdentified,
				Message: "无法识别当前用户，拒绝访问文件系统",
			})
			return
		}
		roots = allowed
		if path == "/" {
			path = allowed[0]
		}
		if !fsutil.IsPathAllowed(path, allowed) {
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
	// 懒猫部署下上级目录若越出允许范围，则停在当前目录（前端据此禁用"上级"按钮）
	if h.cfg.LazyCatDeploy && !fsutil.IsPathAllowed(parent, roots) {
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

// lazycatUsername 取当前懒猫登录用户名。
// LAZYCAT_USERNAME 仅用于开发机直连调试时模拟用户；生产流量经 lzc-ingress，
// 其注入的 X-HC-User-ID 可信（ingress 会先清空客户端伪造的 X-HC-* 再注入）。
func (h *Handler) lazycatUsername(c *gin.Context) string {
	if h.cfg.LazyCatUsername != "" {
		return h.cfg.LazyCatUsername
	}
	return c.GetHeader("X-HC-User-ID")
}

// lazycatAllowedRoots 返回当前用户允许访问的根目录集合：
// 媒体目录 + 该用户自己的文档目录。用户名缺失或含路径成分时拒绝。
func (h *Handler) lazycatAllowedRoots(c *gin.Context) ([]string, error) {
	username := h.lazycatUsername(c)
	if !fsutil.ValidateUsername(username) {
		slog.Warn("拒绝无法识别的懒猫用户", "username", username)
		return nil, fsutil.ErrInvalidUsername
	}
	docRoot := filepath.Join(h.cfg.LazyCatDocumentRoot, username)
	return []string{h.cfg.LazyCatMediaRoot, docRoot}, nil
}
