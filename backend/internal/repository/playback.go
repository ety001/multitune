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

// stringToNullString 将字符串转为 sql.NullString，空字符串为 NULL
func stringToNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

// Save 保存或更新播放状态
func (r *PlaybackRepo) Save(identityID, playlistID, songID string, position int, mode string) (*model.PlaybackState, error) {
	now := time.Now().Unix()
	pl := stringToNullString(playlistID)
	sg := stringToNullString(songID)

	_, err := r.db.Exec(`
		INSERT INTO playback_states (identity_id, playlist_id, song_id, position, mode, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(identity_id) DO UPDATE SET
			playlist_id = excluded.playlist_id,
			song_id = excluded.song_id,
			position = excluded.position,
			mode = excluded.mode,
			updated_at = excluded.updated_at
	`, identityID, pl, sg, position, mode, now)
	if err != nil {
		return nil, fmt.Errorf("保存播放状态失败: %w", err)
	}

	return r.GetByIdentity(identityID)
}

// SaveWithProgress 在单个事务内合并保存播放状态、单曲进度与歌单记忆点。
// 指针参数为 nil 表示保留数据库中的现有值（无现有记录时使用默认值：
// playlist_id/song_id 为 NULL、position 为 0、mode 为 'order'）。
// 通过单条原子 upsert 完成合并，避免读-改-写竞态。
func (r *PlaybackRepo) SaveWithProgress(identityID string, playlistID, songID *string, position *int, mode *string) (*model.PlaybackState, error) {
	now := time.Now().Unix()

	var pl, sg sql.NullString
	if playlistID != nil {
		pl = stringToNullString(*playlistID)
	}
	if songID != nil {
		sg = stringToNullString(*songID)
	}
	pos := 0
	if position != nil {
		pos = *position
	}
	md := "order"
	if mode != nil {
		md = *mode
	}

	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("开始事务失败: %w", err)
	}
	defer tx.Rollback()

	// 原子合并保存播放状态：未传入（nil）的字段保留数据库现有值
	_, err = tx.Exec(`
		INSERT INTO playback_states (identity_id, playlist_id, song_id, position, mode, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(identity_id) DO UPDATE SET
			playlist_id = CASE WHEN ? THEN excluded.playlist_id ELSE playback_states.playlist_id END,
			song_id     = CASE WHEN ? THEN excluded.song_id     ELSE playback_states.song_id     END,
			position    = CASE WHEN ? THEN excluded.position    ELSE playback_states.position    END,
			mode        = CASE WHEN ? THEN excluded.mode        ELSE playback_states.mode        END,
			updated_at  = excluded.updated_at
	`, identityID, pl, sg, pos, md, now,
		playlistID != nil, songID != nil, position != nil, mode != nil)
	if err != nil {
		return nil, fmt.Errorf("保存播放状态失败: %w", err)
	}

	// 同步单曲进度（从合并后的最终状态派生，仅当 song_id 非空）
	_, err = tx.Exec(`
		INSERT INTO song_progress (identity_id, song_id, position, updated_at)
		SELECT identity_id, song_id, position, ?
		FROM playback_states
		WHERE identity_id = ? AND song_id IS NOT NULL
		ON CONFLICT(identity_id, song_id) DO UPDATE SET
			position = excluded.position,
			updated_at = excluded.updated_at
	`, now, identityID)
	if err != nil {
		return nil, fmt.Errorf("保存单曲进度失败: %w", err)
	}

	// 同步歌单记忆点（从合并后的最终状态派生，仅当 playlist_id 非空）
	_, err = tx.Exec(`
		INSERT INTO playlist_states (playlist_id, song_id, position, updated_at)
		SELECT playlist_id, song_id, position, ?
		FROM playback_states
		WHERE identity_id = ? AND playlist_id IS NOT NULL
		ON CONFLICT(playlist_id) DO UPDATE SET
			song_id = excluded.song_id,
			position = excluded.position,
			updated_at = excluded.updated_at
	`, now, identityID)
	if err != nil {
		return nil, fmt.Errorf("保存歌单记忆点失败: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交播放状态事务失败: %w", err)
	}

	return r.GetByIdentity(identityID)
}

// GetPlaylistState 获取歌单播放记忆点
func (r *PlaybackRepo) GetPlaylistState(playlistID string) (*model.PlaylistState, error) {
	var p model.PlaylistState
	var songID sql.NullString
	err := r.db.QueryRow(`
		SELECT playlist_id, song_id, position, updated_at
		FROM playlist_states
		WHERE playlist_id = ?
	`, playlistID).Scan(&p.PlaylistID, &songID, &p.Position, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询歌单记忆点失败: %w", err)
	}
	p.SongID = songID.String
	return &p, nil
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
