package api

import (
	"github.com/ety001/multitune/internal/config"
	"github.com/ety001/multitune/internal/db"
	"github.com/ety001/multitune/internal/repository"
)

// Handler API 处理器
type Handler struct {
	cfg          *config.Config
	db           *db.DB
	identityRepo *repository.IdentityRepo
	playlistRepo *repository.PlaylistRepo
}

// NewHandler 创建处理器
func NewHandler(cfg *config.Config, db *db.DB) *Handler {
	return &Handler{
		cfg:          cfg,
		db:           db,
		identityRepo: repository.NewIdentityRepo(db),
		playlistRepo: repository.NewPlaylistRepo(db),
	}
}
