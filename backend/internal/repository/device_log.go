package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ety001/multitune/internal/db"
	"github.com/ety001/multitune/internal/model"
)

// DeviceLogRepo 设备日志数据访问
type DeviceLogRepo struct {
	db *db.DB
}

// NewDeviceLogRepo 创建设备日志仓库
func NewDeviceLogRepo(database *db.DB) *DeviceLogRepo {
	return &DeviceLogRepo{db: database}
}

// Create 创建设备日志
func (r *DeviceLogRepo) Create(log *model.DeviceLog) (*model.DeviceLog, error) {
	now := time.Now().Unix()
	result, err := r.db.Exec(`
		INSERT INTO device_logs (
			user_agent, chrome_version, webview_version, is_webview,
			screen_width, screen_height, window_width, window_height,
			language, platform, cookie_enabled, online, timestamp, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, log.UserAgent, log.ChromeVersion, log.WebviewVersion, boolToInt(log.IsWebview),
		log.ScreenWidth, log.ScreenHeight, log.WindowWidth, log.WindowHeight,
		log.Language, log.Platform, boolToInt(log.CookieEnabled), boolToInt(log.Online),
		log.Timestamp, now)
	if err != nil {
		return nil, fmt.Errorf("创建设备日志失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("获取设备日志 ID 失败: %w", err)
	}

	return r.GetByID(id)
}

// GetByID 根据 ID 获取设备日志
func (r *DeviceLogRepo) GetByID(id int64) (*model.DeviceLog, error) {
	var log model.DeviceLog
	var userAgent, language, platform, timestamp sql.NullString
	err := r.db.QueryRow(`
		SELECT id, user_agent, chrome_version, webview_version, is_webview,
			screen_width, screen_height, window_width, window_height,
			language, platform, cookie_enabled, online, timestamp, created_at
		FROM device_logs
		WHERE id = ?
	`, id).Scan(
		&log.ID, &userAgent, &log.ChromeVersion, &log.WebviewVersion, &log.IsWebview,
		&log.ScreenWidth, &log.ScreenHeight, &log.WindowWidth, &log.WindowHeight,
		&language, &platform, &log.CookieEnabled, &log.Online, &timestamp, &log.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询设备日志失败: %w", err)
	}

	log.UserAgent = userAgent.String
	log.Language = language.String
	log.Platform = platform.String
	log.Timestamp = timestamp.String
	return &log, nil
}

// List 查询设备日志列表
func (r *DeviceLogRepo) List(limit, offset int) ([]model.DeviceLog, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	var total int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM device_logs").Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计设备日志失败: %w", err)
	}

	rows, err := r.db.Query(`
		SELECT id, user_agent, chrome_version, webview_version, is_webview,
			screen_width, screen_height, window_width, window_height,
			language, platform, cookie_enabled, online, timestamp, created_at
		FROM device_logs
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("查询设备日志列表失败: %w", err)
	}
	defer rows.Close()

	logs := make([]model.DeviceLog, 0)
	for rows.Next() {
		var log model.DeviceLog
		var userAgent, language, platform, timestamp sql.NullString
		if err := rows.Scan(
			&log.ID, &userAgent, &log.ChromeVersion, &log.WebviewVersion, &log.IsWebview,
			&log.ScreenWidth, &log.ScreenHeight, &log.WindowWidth, &log.WindowHeight,
			&language, &platform, &log.CookieEnabled, &log.Online, &timestamp, &log.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("扫描设备日志失败: %w", err)
		}
		log.UserAgent = userAgent.String
		log.Language = language.String
		log.Platform = platform.String
		log.Timestamp = timestamp.String
		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("遍历设备日志列表失败: %w", err)
	}

	return logs, total, nil
}

// boolToInt bool 转 int
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
