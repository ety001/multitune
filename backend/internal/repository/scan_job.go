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

// ScanJobRepo 扫描任务数据访问
type ScanJobRepo struct {
	db *db.DB
}

// NewScanJobRepo 创建扫描任务仓库
func NewScanJobRepo(database *db.DB) *ScanJobRepo {
	return &ScanJobRepo{db: database}
}

// Create 创建扫描任务
func (r *ScanJobRepo) Create(playlistID string) (*model.ScanJob, error) {
	now := time.Now().Unix()
	job := &model.ScanJob{
		ID:         uuid.NewString(),
		PlaylistID: playlistID,
		Status:     "pending",
		Total:      0,
		Current:    0,
		Added:      0,
		Updated:    0,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	_, err := r.db.Exec(`
		INSERT INTO scan_jobs (id, playlist_id, status, total, current, added, updated, message, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, job.ID, job.PlaylistID, job.Status, job.Total, job.Current, job.Added, job.Updated, job.Message, job.CreatedAt, job.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("创建扫描任务失败: %w", err)
	}

	return job, nil
}

// GetByID 根据 ID 获取扫描任务
func (r *ScanJobRepo) GetByID(id string) (*model.ScanJob, error) {
	var job model.ScanJob
	var message sql.NullString
	err := r.db.QueryRow(`
		SELECT id, playlist_id, status, total, current, added, updated, message, created_at, updated_at
		FROM scan_jobs
		WHERE id = ?
	`, id).Scan(&job.ID, &job.PlaylistID, &job.Status, &job.Total, &job.Current, &job.Added, &job.Updated, &message, &job.CreatedAt, &job.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询扫描任务失败: %w", err)
	}
	job.Message = message.String
	return &job, nil
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
