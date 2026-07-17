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

// ErrSongNotInPlaylist 歌曲不在歌单中
var ErrSongNotInPlaylist = errors.New("歌曲不在歌单中")

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

// ContainsSong 检查歌曲是否在歌单中
func (r *PlaylistRepo) ContainsSong(playlistID, songID string) (bool, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM playlist_songs WHERE playlist_id = ? AND song_id = ?
	`, playlistID, songID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("查询歌单歌曲关系失败: %w", err)
	}
	return count > 0, nil
}

// CountSongs 统计歌单内歌曲数量
func (r *PlaylistRepo) CountSongs(playlistID string) (int, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM playlist_songs WHERE playlist_id = ?
	`, playlistID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计歌单歌曲数量失败: %w", err)
	}
	return count, nil
}

// GetMaxSortOrder 获取歌单内最大排序号
func (r *PlaylistRepo) GetMaxSortOrder(playlistID string) (int, error) {
	var maxOrder sql.NullInt64
	err := r.db.QueryRow(`
		SELECT MAX(sort_order) FROM playlist_songs WHERE playlist_id = ?
	`, playlistID).Scan(&maxOrder)
	if err != nil {
		return 0, fmt.Errorf("查询最大排序号失败: %w", err)
	}
	if !maxOrder.Valid {
		return 0, nil
	}
	return int(maxOrder.Int64), nil
}

// AddSongs 添加歌曲到歌单，返回实际新增数量
func (r *PlaylistRepo) AddSongs(playlistID string, songIDs []string) (int, error) {
	if len(songIDs) == 0 {
		return 0, nil
	}

	maxOrder, err := r.GetMaxSortOrder(playlistID)
	if err != nil {
		return 0, err
	}

	now := time.Now().Unix()
	added := 0
	for i, songID := range songIDs {
		result, err := r.db.Exec(`
			INSERT OR IGNORE INTO playlist_songs (playlist_id, song_id, sort_order, created_at)
			VALUES (?, ?, ?, ?)
		`, playlistID, songID, maxOrder+i+1, now)
		if err != nil {
			return added, fmt.Errorf("添加歌曲失败 %s: %w", songID, err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return added, fmt.Errorf("获取插入影响行数失败: %w", err)
		}
		if rows > 0 {
			added++
		}
	}

	return added, nil
}

// RemoveSong 从歌单移除歌曲
func (r *PlaylistRepo) RemoveSong(playlistID, songID string) error {
	_, err := r.db.Exec(`
		DELETE FROM playlist_songs WHERE playlist_id = ? AND song_id = ?
	`, playlistID, songID)
	if err != nil {
		return fmt.Errorf("移除歌曲失败: %w", err)
	}
	return nil
}

// UpdateSongOrder 调整歌单内歌曲顺序
func (r *PlaylistRepo) UpdateSongOrder(playlistID string, songIDs []string) error {
	if len(songIDs) == 0 {
		return nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}
	defer tx.Rollback()

	for i, songID := range songIDs {
		result, err := tx.Exec(`
			UPDATE playlist_songs
			SET sort_order = ?
			WHERE playlist_id = ? AND song_id = ?
		`, i, playlistID, songID)
		if err != nil {
			return fmt.Errorf("更新歌曲顺序失败 %s: %w", songID, err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("获取更新影响行数失败: %w", err)
		}
		if rows == 0 {
			return fmt.Errorf("%w: %s", ErrSongNotInPlaylist, songID)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交顺序更新事务失败: %w", err)
	}

	return nil
}

// ListSongs 获取歌单内歌曲列表（含总数）
func (r *PlaylistRepo) ListSongs(playlistID string, limit, offset int) ([]model.Song, int, error) {
	var total int
	if err := r.db.QueryRow(`
		SELECT COUNT(*) FROM playlist_songs WHERE playlist_id = ?
	`, playlistID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计歌单歌曲失败: %w", err)
	}

	rows, err := r.db.Query(`
		SELECT s.id, s.path, s.source, s.title, s.artist, s.album, s.duration, s.cover_url, s.created_at, s.updated_at,
		       ps.sort_order
		FROM playlist_songs ps
		JOIN songs s ON ps.song_id = s.id
		WHERE ps.playlist_id = ?
		ORDER BY ps.sort_order ASC, ps.created_at ASC
		LIMIT ? OFFSET ?
	`, playlistID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("查询歌单歌曲失败: %w", err)
	}
	defer rows.Close()

	songs := make([]model.Song, 0)
	for rows.Next() {
		var s model.Song
		var artist, album, coverURL sql.NullString
		if err := rows.Scan(&s.ID, &s.Path, &s.Source, &s.Title, &artist, &album, &s.Duration, &coverURL, &s.CreatedAt, &s.UpdatedAt, &s.SortOrder); err != nil {
			return nil, 0, fmt.Errorf("扫描歌曲失败: %w", err)
		}
		s.Artist = artist.String
		s.Album = album.String
		s.CoverURL = coverURL.String
		songs = append(songs, s)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("遍历歌单歌曲失败: %w", err)
	}

	return songs, total, nil
}
