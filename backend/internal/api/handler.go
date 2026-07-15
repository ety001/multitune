package api

import (
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
	scanner       *scanner.Scanner
}

// NewHandler 创建处理器
func NewHandler(cfg *config.Config, db *db.DB) *Handler {
	songRepo := repository.NewSongRepo(db)
	return &Handler{
		cfg:           cfg,
		db:            db,
		identityRepo:  repository.NewIdentityRepo(db),
		playlistRepo:  repository.NewPlaylistRepo(db),
		songRepo:      songRepo,
		playbackRepo:  repository.NewPlaybackRepo(db),
		deviceLogRepo: repository.NewDeviceLogRepo(db),
		scanner:       scanner.New(cfg.MediaRoot, songRepo, cfg.ScanFormats),
	}
}
