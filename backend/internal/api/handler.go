package api

import (
	"log/slog"
	"os"

	"github.com/ety001/multitune/internal/config"
	"github.com/ety001/multitune/internal/db"
	"github.com/ety001/multitune/internal/repository"
	"github.com/ety001/multitune/internal/scanner"
)

// Handler API 处理器
type Handler struct {
	cfg           *config.Config
	db            *db.DB
	identityRepo  *repository.IdentityRepo
	playlistRepo  *repository.PlaylistRepo
	songRepo      *repository.SongRepo
	playbackRepo  *repository.PlaybackRepo
	deviceLogRepo *repository.DeviceLogRepo
	scanJobRepo   *repository.ScanJobRepo
	scanner       *scanner.Scanner
}

// NewHandler 创建处理器
func NewHandler(cfg *config.Config, db *db.DB) *Handler {
	songRepo := repository.NewSongRepo(db)
	// 封面缩略图缓存目录；创建失败不阻止启动，缩略图请求会降级返回原图
	if err := os.MkdirAll(cfg.CachePath, 0755); err != nil {
		slog.Error("创建缓存目录失败，封面缩略图将降级为原图", "error", err, "path", cfg.CachePath)
	}
	return &Handler{
		cfg:           cfg,
		db:            db,
		identityRepo:  repository.NewIdentityRepo(db),
		playlistRepo:  repository.NewPlaylistRepo(db),
		songRepo:      songRepo,
		playbackRepo:  repository.NewPlaybackRepo(db),
		deviceLogRepo: repository.NewDeviceLogRepo(db),
		scanJobRepo:   repository.NewScanJobRepo(db),
		scanner:       scanner.New(songRepo, cfg.ScanFormats),
	}
}
