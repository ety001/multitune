package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ety001/multitune/internal/api"
	"github.com/ety001/multitune/internal/config"
	"github.com/ety001/multitune/internal/db"
)

func main() {
	cfg := config.Load()
	setupLogger(cfg.LogLevel)

	slog.Info("多音盒 MultiTune 后端启动",
		"port", cfg.Port,
		"data_path", cfg.DataPath,
		"media_root", cfg.MediaRoot,
	)

	database, err := db.New(cfg)
	if err != nil {
		slog.Error("数据库初始化失败", "error", err)
		os.Exit(1)
	}

	handler := api.NewHandler(cfg, database)
	r := handler.SetupRouter()

	addr := fmt.Sprintf(":%s", cfg.Port)
	srv := &http.Server{
		Addr:        addr,
		Handler:     r,
		ReadTimeout: 10 * time.Second,
		// WriteTimeout 设为 0：音频流接口需要长时间保持连接传输大文件，
		// 普通 API 接口通过应用层控制响应时间。
		WriteTimeout: 0,
		IdleTimeout:  120 * time.Second,
	}

	// 启动 HTTP 服务
	go func() {
		slog.Info("HTTP 服务启动", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP 服务启动失败", "error", err)
			os.Exit(1)
		}
	}()

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("正在关闭服务...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("服务关闭失败", "error", err)
	}

	if err := database.Close(); err != nil {
		slog.Error("数据库关闭失败", "error", err)
	}

	slog.Info("服务已关闭")
}

func setupLogger(level string) {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})
	slog.SetDefault(slog.New(handler))
}
