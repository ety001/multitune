package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ety001/multitune/internal/db"
	"github.com/ety001/multitune/internal/model"
	"github.com/google/uuid"
)

// PlaylistRepo 歌单数据访问
type PlaylistRepo struct {
	db *db.DB
}

// NewPlaylistRepo 创建歌单仓库
func NewPlaylistRepo(database *db.DB) *PlaylistRepo {
	return &PlaylistRepo{db: database}
}

// ListByIdentity 获取身份下的歌单列表
func (r *PlaylistRepo) ListByIdentity(identityID string) ([]model.Playlist, error) {
	rows, err := r.db.Query(`
		SELECT p.id, p.identity_id, p.name, p.cover_url, p.sort_order, p.created_at, p.updated_at,
		       COUNT(ps.song_id) as song_count
		FROM playlists p
		LEFT JOIN playlist_songs ps ON p.id = ps.playlist_id
		WHERE p.identity_id = ?
		GROUP BY p.id
		ORDER BY p.sort_order ASC, p.created_at ASC
	`, identityID)
	if err != nil {
		return nil, fmt.Errorf("查询歌单列表失败: %w", err)
	}
	defer rows.Close()

	var playlists []model.Playlist
	for rows.Next() {
		var p model.Playlist
		var coverURL sql.NullString
		if err := rows.Scan(&p.ID, &p.IdentityID, &p.Name, &coverURL, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt, &p.SongCount); err != nil {
			return nil, fmt.Errorf("扫描歌单失败: %w", err)
		}
		p.CoverURL = coverURL.String
		playlists = append(playlists, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历歌单列表失败: %w", err)
	}

	return playlists, nil
}

// GetByID 根据 ID 获取歌单
func (r *PlaylistRepo) GetByID(id string) (*model.Playlist, error) {
	var p model.Playlist
	var coverURL sql.NullString
	err := r.db.QueryRow(`
		SELECT p.id, p.identity_id, p.name, p.cover_url, p.sort_order, p.created_at, p.updated_at,
		       COUNT(ps.song_id) as song_count
		FROM playlists p
		LEFT JOIN playlist_songs ps ON p.id = ps.playlist_id
		WHERE p.id = ?
		GROUP BY p.id
	`, id).Scan(&p.ID, &p.IdentityID, &p.Name, &coverURL, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt, &p.SongCount)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询歌单失败: %w", err)
	}
	p.CoverURL = coverURL.String
	return &p, nil
}

// CountByIdentity 统计身份下的歌单数量
func (r *PlaylistRepo) CountByIdentity(identityID string) (int, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM playlists WHERE identity_id = ?
	`, identityID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计歌单数量失败: %w", err)
	}
	return count, nil
}

// Create 创建歌单
func (r *PlaylistRepo) Create(identityID, name string, sortOrder int) (*model.Playlist, error) {
	now := time.Now().Unix()
	id := uuid.NewString()

	_, err := r.db.Exec(`
		INSERT INTO playlists (id, identity_id, name, cover_url, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, NULL, ?, ?, ?)
	`, id, identityID, name, sortOrder, now, now)
	if err != nil {
		return nil, fmt.Errorf("创建歌单失败: %w", err)
	}

	return r.GetByID(id)
}

// Update 更新歌单
func (r *PlaylistRepo) Update(id string, name, coverURL *string, sortOrder *int) (*model.Playlist, error) {
	playlist, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}
	if playlist == nil {
		return nil, nil
	}

	if name != nil {
		playlist.Name = *name
	}
	if coverURL != nil {
		playlist.CoverURL = *coverURL
	}
	if sortOrder != nil {
		playlist.SortOrder = *sortOrder
	}
	playlist.UpdatedAt = time.Now().Unix()

	_, err = r.db.Exec(`
		UPDATE playlists
		SET name = ?, cover_url = ?, sort_order = ?, updated_at = ?
		WHERE id = ?
	`, playlist.Name, playlist.CoverURL, playlist.SortOrder, playlist.UpdatedAt, id)
	if err != nil {
		return nil, fmt.Errorf("更新歌单失败: %w", err)
	}

	return playlist, nil
}

// Delete 删除歌单
func (r *PlaylistRepo) Delete(id string) error {
	result, err := r.db.Exec(`DELETE FROM playlists WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("删除歌单失败: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取删除影响行数失败: %w", err)
	}
	if rows == 0 {
		return nil // 歌单不存在也视为成功删除
	}
	return nil
}
