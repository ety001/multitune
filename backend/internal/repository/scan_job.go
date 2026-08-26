package repository

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ety001/multitune/internal/db"
	"github.com/ety001/multitune/internal/model"
	"github.com/google/uuid"
)

// ScanJobRepo 扫描任务数据访问
type ScanJobRepo struct {
	db *db.DB
}

// NewScanJobRepo 创建扫描任务仓库
func NewScanJobRepo(database *db.DB) *ScanJobRepo {
	return &ScanJobRepo{db: database}
}

// marshalPaths paths 序列化为 JSON 存储（nil 归一为 []）
func marshalPaths(paths []string) (string, error) {
	if paths == nil {
		paths = []string{}
	}
	b, err := json.Marshal(paths)
	if err != nil {
		return "", fmt.Errorf("序列化扫描路径失败: %w", err)
	}
	return string(b), nil
}

// unmarshalPaths 从 JSON 还原 paths，解析失败时降级为空列表
func unmarshalPaths(raw string) []string {
	paths := make([]string, 0)
	if raw == "" {
		return paths
	}
	_ = json.Unmarshal([]byte(raw), &paths)
	return paths
}

// Create 创建扫描任务
func (r *ScanJobRepo) Create(playlistID string, paths []string) (*model.ScanJob, error) {
	now := time.Now().Unix()
	job := &model.ScanJob{
		ID:         uuid.NewString(),
		PlaylistID: playlistID,
		Paths:      paths,
		Status:     "pending",
		Total:      0,
		Current:    0,
		Added:      0,
		Updated:    0,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	pathsJSON, err := marshalPaths(job.Paths)
	if err != nil {
		return nil, err
	}

	_, err = r.db.Exec(`
		INSERT INTO scan_jobs (id, playlist_id, paths, status, total, current, added, updated, message, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, job.ID, job.PlaylistID, pathsJSON, job.Status, job.Total, job.Current, job.Added, job.Updated, job.Message, job.CreatedAt, job.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("创建扫描任务失败: %w", err)
	}

	return job, nil
}

// GetByID 根据 ID 获取扫描任务
func (r *ScanJobRepo) GetByID(id string) (*model.ScanJob, error) {
	var job model.ScanJob
	var pathsJSON string
	var message sql.NullString
	err := r.db.QueryRow(`
		SELECT id, playlist_id, paths, status, total, current, added, updated, message, created_at, updated_at
		FROM scan_jobs
		WHERE id = ?
	`, id).Scan(&job.ID, &job.PlaylistID, &pathsJSON, &job.Status, &job.Total, &job.Current, &job.Added, &job.Updated, &message, &job.CreatedAt, &job.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询扫描任务失败: %w", err)
	}
	job.Message = message.String
	job.Paths = unmarshalPaths(pathsJSON)
	return &job, nil
}

// List 按创建时间倒序列出扫描任务（附目标歌单名），limit 上限 50
func (r *ScanJobRepo) List(limit int) ([]model.ScanJobWithPlaylist, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	rows, err := r.db.Query(`
		SELECT j.id, j.playlist_id, j.paths, j.status, j.total, j.current,
		       j.added, j.updated, j.message, j.created_at, j.updated_at,
		       COALESCE(p.name, '')
		FROM scan_jobs j
		LEFT JOIN playlists p ON p.id = j.playlist_id
		ORDER BY j.created_at DESC, j.rowid DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("查询扫描任务列表失败: %w", err)
	}
	defer rows.Close()

	jobs := make([]model.ScanJobWithPlaylist, 0)
	for rows.Next() {
		var job model.ScanJobWithPlaylist
		var pathsJSON string
		var message sql.NullString
		if err := rows.Scan(&job.ID, &job.PlaylistID, &pathsJSON, &job.Status, &job.Total, &job.Current,
			&job.Added, &job.Updated, &message, &job.CreatedAt, &job.UpdatedAt, &job.PlaylistName); err != nil {
			return nil, fmt.Errorf("读取扫描任务行失败: %w", err)
		}
		job.Message = message.String
		job.Paths = unmarshalPaths(pathsJSON)
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历扫描任务失败: %w", err)
	}

	return jobs, nil
}

// Update 更新扫描任务
func (r *ScanJobRepo) Update(job *model.ScanJob) error {
	job.UpdatedAt = time.Now().Unix()
	_, err := r.db.Exec(`
		UPDATE scan_jobs
		SET status = ?, total = ?, current = ?, added = ?, updated = ?, message = ?, updated_at = ?
		WHERE id = ?
	`, job.Status, job.Total, job.Current, job.Added, job.Updated, job.Message, job.UpdatedAt, job.ID)
	if err != nil {
		return fmt.Errorf("更新扫描任务失败: %w", err)
	}
	return nil
}
