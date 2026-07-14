package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ety001/multitune/internal/config"
	_ "modernc.org/sqlite"
)

// DB 数据库封装
type DB struct {
	*sql.DB
	cfg *config.Config
}

// New 创建数据库连接
func New(cfg *config.Config) (*DB, error) {
	if err := os.MkdirAll(cfg.DataPath, 0755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}

	dbPath := filepath.Join(cfg.DataPath, cfg.DatabaseName)
	dsn := fmt.Sprintf("%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", dbPath)

	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	db := &DB{DB: sqlDB, cfg: cfg}
	if err := db.Migrate(); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	return db, nil
}

// Close 关闭数据库连接
func (db *DB) Close() error {
	return db.DB.Close()
}
