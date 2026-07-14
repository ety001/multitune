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

// Upsert 插入或更新歌曲
func (r *SongRepo) Upsert(path, source, title, artist, album string, duration int) (*UpsertResult, error) {
	now := time.Now().Unix()

	existing, err := r.GetByPath(path)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		_, err := r.db.Exec(`
			UPDATE songs
			SET source = ?, title = ?, artist = ?, album = ?, duration = ?, cover_url = NULL, updated_at = ?
			WHERE id = ?
		`, source, title, artist, album, duration, now, existing.ID)
		if err != nil {
			return nil, fmt.Errorf("更新歌曲失败: %w", err)
		}
		song, err := r.GetByID(existing.ID)
		if err != nil {
			return nil, err
		}
		return &UpsertResult{Song: song, IsNew: false}, nil
	}

	id := uuid.NewString()
	_, err = r.db.Exec(`
		INSERT INTO songs (id, path, source, title, artist, album, duration, cover_url, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, NULL, ?, ?)
	`, id, path, source, title, artist, album, duration, now, now)
	if err != nil {
		return nil, fmt.Errorf("插入歌曲失败: %w", err)
	}
	song, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}
	return &UpsertResult{Song: song, IsNew: true}, nil
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

	var songs []model.Song
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
