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

// IdentityRepo 身份数据访问
type IdentityRepo struct {
	db *db.DB
}

// NewIdentityRepo 创建身份仓库
func NewIdentityRepo(database *db.DB) *IdentityRepo {
	return &IdentityRepo{db: database}
}

// List 获取身份列表（按 sort_order 升序）
func (r *IdentityRepo) List() ([]model.Identity, error) {
	rows, err := r.db.Query(`
		SELECT id, name, avatar_color, sort_order, is_default, created_at, updated_at
		FROM identities
		ORDER BY sort_order ASC, created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("查询身份列表失败: %w", err)
	}
	defer rows.Close()

	var identities []model.Identity
	for rows.Next() {
		var i model.Identity
		var isDefault int
		if err := rows.Scan(&i.ID, &i.Name, &i.AvatarColor, &i.SortOrder, &isDefault, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, fmt.Errorf("扫描身份失败: %w", err)
		}
		i.IsDefault = isDefault == 1
		identities = append(identities, i)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历身份列表失败: %w", err)
	}

	return identities, nil
}

// GetByID 根据 ID 获取身份
func (r *IdentityRepo) GetByID(id string) (*model.Identity, error) {
	var i model.Identity
	var isDefault int
	err := r.db.QueryRow(`
		SELECT id, name, avatar_color, sort_order, is_default, created_at, updated_at
		FROM identities
		WHERE id = ?
	`, id).Scan(&i.ID, &i.Name, &i.AvatarColor, &i.SortOrder, &isDefault, &i.CreatedAt, &i.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询身份失败: %w", err)
	}
	i.IsDefault = isDefault == 1
	return &i, nil
}

// Count 获取身份总数
func (r *IdentityRepo) Count() (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM identities`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计身份数量失败: %w", err)
	}
	return count, nil
}

// Create 创建身份
func (r *IdentityRepo) Create(name, avatarColor string, sortOrder int) (*model.Identity, error) {
	now := time.Now().Unix()
	id := uuid.NewString()

	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("开始事务失败: %w", err)
	}
	defer tx.Rollback()

	// 先以非默认状态插入，避免并发时违反唯一默认身份约束
	if _, err := tx.Exec(`
		INSERT INTO identities (id, name, avatar_color, sort_order, is_default, created_at, updated_at)
		VALUES (?, ?, ?, ?, 0, ?, ?)
	`, id, name, avatarColor, sortOrder, now, now); err != nil {
		return nil, fmt.Errorf("创建身份失败: %w", err)
	}

	// 如果当前没有其他默认身份，则将新身份设为默认
	var defaultCount int
	if err := tx.QueryRow(`
		SELECT COUNT(*) FROM identities WHERE is_default = 1 AND id != ?
	`, id).Scan(&defaultCount); err != nil {
		return nil, fmt.Errorf("检查默认身份失败: %w", err)
	}
	if defaultCount == 0 {
		if _, err := tx.Exec(`
			UPDATE identities SET is_default = 1, updated_at = ? WHERE id = ?
		`, now, id); err != nil {
			return nil, fmt.Errorf("设置默认身份失败: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交创建事务失败: %w", err)
	}

	return r.GetByID(id)
}

// Update 更新身份
func (r *IdentityRepo) Update(id string, name, avatarColor *string, sortOrder *int) (*model.Identity, error) {
	identity, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}
	if identity == nil {
		return nil, nil
	}

	if name != nil {
		identity.Name = *name
	}
	if avatarColor != nil {
		identity.AvatarColor = *avatarColor
	}
	if sortOrder != nil {
		identity.SortOrder = *sortOrder
	}
	identity.UpdatedAt = time.Now().Unix()

	var isDefault int
	if identity.IsDefault {
		isDefault = 1
	}

	_, err = r.db.Exec(`
		UPDATE identities
		SET name = ?, avatar_color = ?, sort_order = ?, is_default = ?, updated_at = ?
		WHERE id = ?
	`, identity.Name, identity.AvatarColor, identity.SortOrder, isDefault, identity.UpdatedAt, id)
	if err != nil {
		return nil, fmt.Errorf("更新身份失败: %w", err)
	}

	return identity, nil
}

// Delete 删除身份
func (r *IdentityRepo) Delete(id string) error {
	identity, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if identity == nil {
		return nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}
	defer tx.Rollback()

	// 级联删除由数据库外键处理，这里手动处理默认身份转移
	if _, err := tx.Exec(`DELETE FROM identities WHERE id = ?`, id); err != nil {
		return fmt.Errorf("删除身份失败: %w", err)
	}

	if identity.IsDefault {
		var nextID string
		err := tx.QueryRow(`
			SELECT id FROM identities
			ORDER BY sort_order ASC, created_at ASC
			LIMIT 1
		`).Scan(&nextID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("查询下一个默认身份失败: %w", err)
		}
		if err == nil {
			if _, err := tx.Exec(`
				UPDATE identities SET is_default = 1, updated_at = ? WHERE id = ?
			`, time.Now().Unix(), nextID); err != nil {
				return fmt.Errorf("转移默认身份失败: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交删除事务失败: %w", err)
	}

	return nil
}

// SetDefault 设置默认身份
func (r *IdentityRepo) SetDefault(id string) (*model.Identity, error) {
	identity, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}
	if identity == nil {
		return nil, nil
	}
	if identity.IsDefault {
		return identity, nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("开始事务失败: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().Unix()

	// 取消其他默认身份
	if _, err := tx.Exec(`
		UPDATE identities SET is_default = 0, updated_at = ? WHERE is_default = 1
	`, now); err != nil {
		return nil, fmt.Errorf("取消默认身份失败: %w", err)
	}

	// 设置当前为默认
	if _, err := tx.Exec(`
		UPDATE identities SET is_default = 1, updated_at = ? WHERE id = ?
	`, now, id); err != nil {
		return nil, fmt.Errorf("设置默认身份失败: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交默认身份事务失败: %w", err)
	}

	identity.IsDefault = true
	identity.UpdatedAt = now
	return identity, nil
}
