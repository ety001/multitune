package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
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

	playlists := make([]model.Playlist, 0)
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

// Update 更新歌单（单条 SQL，避免 read-modify-write 竞态）
func (r *PlaylistRepo) Update(id string, name, coverURL *string, sortOrder *int) (*model.Playlist, error) {
	// 先检查是否存在
	existing, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, nil
	}

	now := time.Now().Unix()

	// 动态拼接 SET 子句，只更新传入的字段
	setParts := []string{"updated_at = ?"}
	args := []interface{}{now}
	if name != nil {
		setParts = append(setParts, "name = ?")
		args = append(args, *name)
	}
	if coverURL != nil {
		setParts = append(setParts, "cover_url = ?")
		args = append(args, *coverURL)
	}
	if sortOrder != nil {
		setParts = append(setParts, "sort_order = ?")
		args = append(args, *sortOrder)
	}
	args = append(args, id)

	query := "UPDATE playlists SET " + strings.Join(setParts, ", ") + " WHERE id = ?"
	if _, err := r.db.Exec(query, args...); err != nil {
		return nil, fmt.Errorf("更新歌单失败: %w", err)
	}

	return r.GetByID(id)
}

// Delete 删除歌单，返回是否实际删除了记录
func (r *PlaylistRepo) Delete(id string) (bool, error) {
	result, err := r.db.Exec(`DELETE FROM playlists WHERE id = ?`, id)
	if err != nil {
		return false, fmt.Errorf("删除歌单失败: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("获取删除影响行数失败: %w", err)
	}
	return rows > 0, nil
}
