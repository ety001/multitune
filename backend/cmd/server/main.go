package main

import (
	"fmt"
	"log/slog"
	"os"

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
	defer database.Close()

	handler := api.NewHandler(cfg, database)
	r := handler.SetupRouter()

	addr := fmt.Sprintf(":%s", cfg.Port)
	slog.Info("HTTP 服务启动", "addr", addr)
	if err := r.Run(addr); err != nil {
		slog.Error("HTTP 服务启动失败", "error", err)
		os.Exit(1)
	}
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
