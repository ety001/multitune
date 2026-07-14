package api

import (
	"log/slog"
	"net/http"
	"os"
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
		api.GET("/songs", h.ListSongs)
		api.GET("/songs/:id", h.GetSong)

		// 播放状态 API
		api.GET("/playback/:identityId", h.GetPlaybackState)
		api.POST("/playback/:identityId", h.SavePlaybackState)
	}

	// 静态文件服务
	staticPath := h.cfg.StaticPath
	if _, err := os.Stat(staticPath); err == nil {
		r.Static("/", staticPath)
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
