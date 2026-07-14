package db

import (
	"embed"
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Migrate 执行数据库迁移
func (db *DB) Migrate() error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS _migrations (
		version INTEGER PRIMARY KEY,
		applied_at INTEGER NOT NULL
	)`); err != nil {
		return fmt.Errorf("创建迁移表失败: %w", err)
	}

	currentVersion, err := db.getCurrentVersion()
	if err != nil {
		return err
	}

	files, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("读取迁移文件失败: %w", err)
	}

	var migrations []migrationFile
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".sql") {
			continue
		}
		version, err := parseVersion(f.Name())
		if err != nil {
			return err
		}
		migrations = append(migrations, migrationFile{
			version: version,
			name:    f.Name(),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].version < migrations[j].version
	})

	for _, m := range migrations {
		if m.version <= currentVersion {
			continue
		}

		sql, err := migrationFS.ReadFile(filepath.Join("migrations", m.name))
		if err != nil {
			return fmt.Errorf("读取迁移文件 %s 失败: %w", m.name, err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("开始事务失败: %w", err)
		}

		if _, err := tx.Exec(string(sql)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("执行迁移 %s 失败: %w", m.name, err)
		}

		if _, err := tx.Exec(
			"INSERT INTO _migrations (version, applied_at) VALUES (?, ?)",
			m.version, time.Now().Unix(),
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("记录迁移版本失败: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("提交迁移事务失败: %w", err)
		}

		slog.Info("数据库迁移执行成功", "version", m.version, "file", m.name)
	}

	return nil
}

func (db *DB) getCurrentVersion() (int, error) {
	var version int
	err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM _migrations").Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("查询当前迁移版本失败: %w", err)
	}
	return version, nil
}

type migrationFile struct {
	version int
	name    string
}

func parseVersion(name string) (int, error) {
	parts := strings.Split(name, "_")
	if len(parts) < 2 {
		return 0, fmt.Errorf("非法的迁移文件名: %s", name)
	}
	version, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("解析迁移版本失败 %s: %w", name, err)
	}
	return version, nil
}
