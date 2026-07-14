package repository

import (
	"testing"

	"github.com/ety001/multitune/internal/config"
	"github.com/ety001/multitune/internal/db"
)

func newTestDB(t *testing.T) *db.DB {
	t.Helper()
	cfg := &config.Config{
		DataPath:                t.TempDir(),
		DatabaseName:            "test.db",
		MaxIdentities:           20,
		MaxPlaylistsPerIdentity: 50,
		MaxSongsPerPlaylist:     1000,
	}
	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	t.Cleanup(func() {
		_ = database.Close()
	})
	return database
}

func TestIdentityRepo_Create(t *testing.T) {
	database := newTestDB(t)
	repo := NewIdentityRepo(database)

	identity, err := repo.Create("爸爸", "#6366f1", 0)
	if err != nil {
		t.Fatalf("创建身份失败: %v", err)
	}
	if identity == nil {
		t.Fatal("身份不应为空")
	}
	if identity.Name != "爸爸" {
		t.Errorf("名称错误: got %s, want 爸爸", identity.Name)
	}
	if !identity.IsDefault {
		t.Error("第一个身份应自动设为默认")
	}

	// 第二个身份不应为默认
	identity2, err := repo.Create("妈妈", "#ec4899", 1)
	if err != nil {
		t.Fatalf("创建第二个身份失败: %v", err)
	}
	if identity2.IsDefault {
		t.Error("第二个身份不应为默认")
	}
}

func TestIdentityRepo_List(t *testing.T) {
	database := newTestDB(t)
	repo := NewIdentityRepo(database)

	if _, err := repo.Create("爸爸", "#6366f1", 0); err != nil {
		t.Fatalf("创建身份失败: %v", err)
	}
	if _, err := repo.Create("妈妈", "#ec4899", 1); err != nil {
		t.Fatalf("创建身份失败: %v", err)
	}

	identities, err := repo.List()
	if err != nil {
		t.Fatalf("获取身份列表失败: %v", err)
	}
	if len(identities) != 2 {
		t.Errorf("身份数量错误: got %d, want 2", len(identities))
	}
	if identities[0].Name != "爸爸" {
		t.Errorf("排序错误: got %s, want 爸爸", identities[0].Name)
	}
}

func TestIdentityRepo_SetDefault(t *testing.T) {
	database := newTestDB(t)
	repo := NewIdentityRepo(database)

	identity1, _ := repo.Create("爸爸", "#6366f1", 0)
	identity2, _ := repo.Create("妈妈", "#ec4899", 1)

	updated, err := repo.SetDefault(identity2.ID)
	if err != nil {
		t.Fatalf("设置默认身份失败: %v", err)
	}
	if !updated.IsDefault {
		t.Error("妈妈应被设为默认")
	}

	// 检查原默认身份已被取消
	old, err := repo.GetByID(identity1.ID)
	if err != nil {
		t.Fatalf("获取身份失败: %v", err)
	}
	if old.IsDefault {
		t.Error("爸爸的默认身份应被取消")
	}
}

func TestIdentityRepo_Delete(t *testing.T) {
	database := newTestDB(t)
	repo := NewIdentityRepo(database)

	identity1, _ := repo.Create("爸爸", "#6366f1", 0)
	identity2, _ := repo.Create("妈妈", "#ec4899", 1)

	if err := repo.Delete(identity1.ID); err != nil {
		t.Fatalf("删除身份失败: %v", err)
	}

	remaining, _ := repo.List()
	if len(remaining) != 1 {
		t.Fatalf("剩余身份数量错误: got %d, want 1", len(remaining))
	}
	if !remaining[0].IsDefault {
		t.Error("删除默认身份后，剩余身份应自动设为默认")
	}
	if remaining[0].ID != identity2.ID {
		t.Errorf("默认身份转移错误: got %s, want %s", remaining[0].ID, identity2.ID)
	}
}

func TestIdentityRepo_Count(t *testing.T) {
	database := newTestDB(t)
	repo := NewIdentityRepo(database)

	count, err := repo.Count()
	if err != nil {
		t.Fatalf("统计身份失败: %v", err)
	}
	if count != 0 {
		t.Errorf("初始数量错误: got %d, want 0", count)
	}

	if _, err := repo.Create("测试", "#000000", 0); err != nil {
		t.Fatalf("创建身份失败: %v", err)
	}

	count, err = repo.Count()
	if err != nil {
		t.Fatalf("统计身份失败: %v", err)
	}
	if count != 1 {
		t.Errorf("创建后数量错误: got %d, want 1", count)
	}
}

func TestIdentityRepo_MaxLimit(t *testing.T) {
	database := newTestDB(t)
	repo := NewIdentityRepo(database)

	for i := 0; i < 20; i++ {
		if _, err := repo.Create("身份", "#000000", i); err != nil {
			t.Fatalf("创建身份失败: %v", err)
		}
	}

	count, _ := repo.Count()
	if count != 20 {
		t.Errorf("身份数量错误: got %d, want 20", count)
	}
}
