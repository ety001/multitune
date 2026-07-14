package api

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// SetupRouter 配置路由
func (h *Handler) SetupRouter() *gin.Engine {
	mode := os.Getenv("GIN_MODE")
	if mode == "" {
		mode = gin.ReleaseMode
	}
	gin.SetMode(mode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(corsMiddleware())
	r.Use(requestLogger())

	// API 路由
	api := r.Group("/api")
	{
		api.GET("/healthz", h.HealthCheck)
	}

	// 静态文件服务
	staticPath := os.Getenv("STATIC_PATH")
	if staticPath == "" {
		staticPath = "/app/static"
	}
	if _, err := os.Stat(staticPath); err == nil {
		r.Static("/", staticPath)
	}

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
			" | " + fmt.Sprintf("%d", param.StatusCode) +
			" | " + param.Latency.String() +
			" | " + param.Method +
			" | " + param.Path +
			"\n"
	})
}
