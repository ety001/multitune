package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ety001/multitune/internal/db"
	"github.com/ety001/multitune/internal/model"
)

// PlaybackRepo 播放状态数据访问
type PlaybackRepo struct {
	db *db.DB
}

// NewPlaybackRepo 创建播放状态仓库
func NewPlaybackRepo(database *db.DB) *PlaybackRepo {
	return &PlaybackRepo{db: database}
}

// GetByIdentity 获取身份播放状态
func (r *PlaybackRepo) GetByIdentity(identityID string) (*model.PlaybackState, error) {
	var p model.PlaybackState
	var playlistID, songID sql.NullString
	err := r.db.QueryRow(`
		SELECT identity_id, playlist_id, song_id, position, mode, updated_at
		FROM playback_states
		WHERE identity_id = ?
	`, identityID).Scan(&p.IdentityID, &playlistID, &songID, &p.Position, &p.Mode, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询播放状态失败: %w", err)
	}
	p.PlaylistID = playlistID.String
	p.SongID = songID.String
	return &p, nil
}

// Save 保存或更新播放状态
func (r *PlaybackRepo) Save(identityID, playlistID, songID string, position int, mode string) (*model.PlaybackState, error) {
	now := time.Now().Unix()

	_, err := r.db.Exec(`
		INSERT INTO playback_states (identity_id, playlist_id, song_id, position, mode, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(identity_id) DO UPDATE SET
			playlist_id = excluded.playlist_id,
			song_id = excluded.song_id,
			position = excluded.position,
			mode = excluded.mode,
			updated_at = excluded.updated_at
	`, identityID, playlistID, songID, position, mode, now)
	if err != nil {
		return nil, fmt.Errorf("保存播放状态失败: %w", err)
	}

	return r.GetByIdentity(identityID)
}

// SaveSongProgress 保存单曲进度
func (r *PlaybackRepo) SaveSongProgress(identityID, songID string, position int) error {
	now := time.Now().Unix()
	_, err := r.db.Exec(`
		INSERT INTO song_progress (identity_id, song_id, position, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(identity_id, song_id) DO UPDATE SET
			position = excluded.position,
			updated_at = excluded.updated_at
	`, identityID, songID, position, now)
	if err != nil {
		return fmt.Errorf("保存单曲进度失败: %w", err)
	}
	return nil
}

// GetSongProgress 获取单曲进度
func (r *PlaybackRepo) GetSongProgress(identityID, songID string) (int, error) {
	var position int
	err := r.db.QueryRow(`
		SELECT position FROM song_progress WHERE identity_id = ? AND song_id = ?
	`, identityID, songID).Scan(&position)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("查询单曲进度失败: %w", err)
	}
	return position, nil
}

// Delete 删除播放状态
func (r *PlaybackRepo) Delete(identityID string) error {
	_, err := r.db.Exec(`DELETE FROM playback_states WHERE identity_id = ?`, identityID)
	if err != nil {
		return fmt.Errorf("删除播放状态失败: %w", err)
	}
	return nil
}
