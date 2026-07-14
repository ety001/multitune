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

// SongRepo 歌曲数据访问
type SongRepo struct {
	db *db.DB
}

// NewSongRepo 创建歌曲仓库
func NewSongRepo(database *db.DB) *SongRepo {
	return &SongRepo{db: database}
}

// GetByID 根据 ID 获取歌曲
func (r *SongRepo) GetByID(id string) (*model.Song, error) {
	var s model.Song
	var artist, album, coverURL sql.NullString
	err := r.db.QueryRow(`
		SELECT id, path, source, title, artist, album, duration, cover_url, created_at, updated_at
		FROM songs
		WHERE id = ?
	`, id).Scan(&s.ID, &s.Path, &s.Source, &s.Title, &artist, &album, &s.Duration, &coverURL, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询歌曲失败: %w", err)
	}
	s.Artist = artist.String
	s.Album = album.String
	s.CoverURL = coverURL.String
	return &s, nil
}

// GetByPath 根据路径获取歌曲
func (r *SongRepo) GetByPath(path string) (*model.Song, error) {
	var s model.Song
	var artist, album, coverURL sql.NullString
	err := r.db.QueryRow(`
		SELECT id, path, source, title, artist, album, duration, cover_url, created_at, updated_at
		FROM songs
		WHERE path = ?
	`, path).Scan(&s.ID, &s.Path, &s.Source, &s.Title, &artist, &album, &s.Duration, &coverURL, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询歌曲失败: %w", err)
	}
	s.Artist = artist.String
	s.Album = album.String
	s.CoverURL = coverURL.String
	return &s, nil
}

// UpsertResult Upsert 操作结果
type UpsertResult struct {
	Song  *model.Song
	IsNew bool
}

// Upsert 插入或更新歌曲（原子操作，避免 read-modify-write 竞态）
func (r *SongRepo) Upsert(path, source, title, artist, album string, duration int) (*UpsertResult, error) {
	now := time.Now().Unix()
	id := uuid.NewString()

	// 先尝试插入，利用 path UNIQUE 约束
	result, err := r.db.Exec(`
		INSERT OR IGNORE INTO songs (id, path, source, title, artist, album, duration, cover_url, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, NULL, ?, ?)
	`, id, path, source, title, artist, album, duration, now, now)
	if err != nil {
		return nil, fmt.Errorf("保存歌曲失败: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("获取影响行数失败: %w", err)
	}

	if rows > 0 {
		// 新插入成功
		song, err := r.GetByID(id)
		if err != nil {
			return nil, err
		}
		if song == nil {
			return nil, fmt.Errorf("插入后查询失败: id=%s", id)
		}
		return &UpsertResult{Song: song, IsNew: true}, nil
	}

	// 插入被忽略（path 冲突），执行更新（不碰 cover_url）
	_, err = r.db.Exec(`
		UPDATE songs
		SET source = ?, title = ?, artist = ?, album = ?, duration = ?, updated_at = ?
		WHERE path = ?
	`, source, title, artist, album, duration, now, path)
	if err != nil {
		return nil, fmt.Errorf("更新歌曲失败: %w", err)
	}

	song, err := r.GetByPath(path)
	if err != nil {
		return nil, err
	}
	if song == nil {
		return nil, fmt.Errorf("更新后查询失败: path=%s", path)
	}
	return &UpsertResult{Song: song, IsNew: false}, nil
}

// List 歌曲列表/搜索
func (r *SongRepo) List(query, source string, limit, offset int) ([]model.Song, int, error) {
	whereClauses := []string{"1=1"}
	args := []interface{}{}

	if query != "" {
		whereClauses = append(whereClauses, "(title LIKE ? OR artist LIKE ? OR album LIKE ?)")
		like := "%" + query + "%"
		args = append(args, like, like, like)
	}
	if source != "" {
		whereClauses = append(whereClauses, "source = ?")
		args = append(args, source)
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	var total int
	countArgs := append([]interface{}{}, args...)
	if err := r.db.QueryRow(
		"SELECT COUNT(*) FROM songs WHERE "+whereSQL,
		countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计歌曲数量失败: %w", err)
	}

	queryArgs := append([]interface{}{}, args...)
	queryArgs = append(queryArgs, limit, offset)
	rows, err := r.db.Query(`
		SELECT id, path, source, title, artist, album, duration, cover_url, created_at, updated_at
		FROM songs
		WHERE `+whereSQL+`
		ORDER BY title ASC
		LIMIT ? OFFSET ?
	`, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询歌曲列表失败: %w", err)
	}
	defer rows.Close()

	songs := make([]model.Song, 0)
	for rows.Next() {
		var s model.Song
		var artist, album, coverURL sql.NullString
		if err := rows.Scan(&s.ID, &s.Path, &s.Source, &s.Title, &artist, &album, &s.Duration, &coverURL, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("扫描歌曲失败: %w", err)
		}
		s.Artist = artist.String
		s.Album = album.String
		s.CoverURL = coverURL.String
		songs = append(songs, s)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("遍历歌曲列表失败: %w", err)
	}

	return songs, total, nil
}

// Delete 删除歌曲
func (r *SongRepo) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM songs WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("删除歌曲失败: %w", err)
	}
	return nil
}

// CountByIDs 批量校验歌曲 ID 存在性，返回存在的数量
func (r *SongRepo) CountByIDs(ids []string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	query := "SELECT COUNT(*) FROM songs WHERE id IN (" + strings.Join(placeholders, ",") + ")"
	var count int
	if err := r.db.QueryRow(query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("批量校验歌曲失败: %w", err)
	}
	return count, nil
}
