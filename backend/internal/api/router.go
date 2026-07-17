package api

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/ety001/multitune/internal/model"
	"github.com/gin-gonic/gin"
)

// SetupRouter 配置路由
func (h *Handler) SetupRouter() *gin.Engine {
	gin.SetMode(h.cfg.GINMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(corsMiddleware())
	r.Use(requestLogger())

	// API 路由
	api := r.Group("/api")
	{
		api.GET("/healthz", h.HealthCheck)

		// 设备信息日志 API
		api.POST("/device-info", h.CreateDeviceLog)
		api.GET("/device-logs", h.ListDeviceLogs)

		// 身份 API
		api.GET("/identities", h.ListIdentities)
		api.POST("/identities", h.CreateIdentity)
		api.GET("/identities/:id", h.GetIdentity)
		api.PUT("/identities/:id", h.UpdateIdentity)
		api.DELETE("/identities/:id", h.DeleteIdentity)
		api.POST("/identities/:id/default", h.SetDefaultIdentity)

		// 歌单 API
		api.GET("/identities/:id/playlists", h.ListPlaylists)
		api.POST("/identities/:id/playlists", h.CreatePlaylist)
		api.GET("/playlists/:id", h.GetPlaylist)
		api.PUT("/playlists/:id", h.UpdatePlaylist)
		api.DELETE("/playlists/:id", h.DeletePlaylist)
		api.POST("/playlists/:id/songs", h.AddSongsToPlaylist)
		api.DELETE("/playlists/:id/songs/:songId", h.RemoveSongFromPlaylist)
		api.PUT("/playlists/:id/songs/order", h.UpdatePlaylistSongOrder)

		// 文件浏览器 API
		api.GET("/fs/sources", h.ListStorageSources)
		api.GET("/fs/list", h.ListDirectory)
		api.GET("/fs/search", h.SearchSongs)

		// 歌曲与扫描 API
		api.POST("/scan", h.ScanSongs)
		api.POST("/scan/jobs", h.CreateScanJob)
		api.GET("/scan/jobs/:id", h.GetScanJob)
		api.GET("/songs", h.ListSongs)
		api.GET("/songs/:id", h.GetSong)
		api.GET("/songs/:id/stream", h.StreamSong)

		// 播放状态 API
		api.GET("/playback/:identityId", h.GetPlaybackState)
		api.POST("/playback/:identityId", h.SavePlaybackState)
		api.GET("/playlists/:id/progress", h.GetPlaylistProgress)
	}

	// 静态文件服务（避免根路径通配与 /api 冲突，分别挂载子目录）
	staticPath := h.cfg.StaticPath
	if info, err := os.Stat(staticPath); err == nil && info.IsDir() {
		carPath := filepath.Join(staticPath, "car")
		fullPath := filepath.Join(staticPath, "full")
		indexPath := filepath.Join(staticPath, "index.html")

		if _, err := os.Stat(carPath); err == nil {
			r.Static("/car", carPath)
		}
		if _, err := os.Stat(fullPath); err == nil {
			r.Static("/full", fullPath)
		}
		if _, err := os.Stat(indexPath); err == nil {
			r.GET("/", serveIndex(indexPath))
		}
		logoPath := filepath.Join(staticPath, "logo.png")
		if _, err := os.Stat(logoPath); err == nil {
			r.StaticFile("/logo.png", logoPath)
		}
		faviconPath := filepath.Join(staticPath, "favicon.png")
		if _, err := os.Stat(faviconPath); err == nil {
			r.StaticFile("/favicon.png", faviconPath)
			r.GET("/favicon.ico", func(c *gin.Context) {
				c.Redirect(http.StatusMovedPermanently, "/favicon.png")
			})
		}
	} else {
		slog.Warn("静态文件目录不存在，仅提供 API 服务", "path", staticPath)
	}

	// 404/405 统一响应
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, model.APIResponse{
			Code:    404,
			Message: "接口不存在",
		})
	})
	r.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, model.APIResponse{
			Code:    405,
			Message: "请求方法不允许",
		})
	})

	return r
}

// serveIndex 返回入口页文件
func serveIndex(indexPath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.File(indexPath)
	}
}

// corsMiddleware 跨域中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// requestLogger 简单请求日志
func requestLogger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return param.TimeStamp.Format("2006-01-02 15:04:05") +
			" | " + strconv.Itoa(param.StatusCode) +
			" | " + param.Latency.String() +
			" | " + param.Method +
			" | " + param.Path +
			"\n"
	})
}
