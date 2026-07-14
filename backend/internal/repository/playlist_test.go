package repository

import (
	"testing"
)

func TestPlaylistRepo_CreateAndList(t *testing.T) {
	database := newTestDB(t)
	identityRepo := NewIdentityRepo(database)
	playlistRepo := NewPlaylistRepo(database)

	identity, err := identityRepo.Create("爸爸", "#6366f1", 0)
	if err != nil {
		t.Fatalf("创建身份失败: %v", err)
	}

	playlist, err := playlistRepo.Create(identity.ID, "通勤", 0)
	if err != nil {
		t.Fatalf("创建歌单失败: %v", err)
	}
	if playlist == nil {
		t.Fatal("歌单不应为空")
	}
	if playlist.Name != "通勤" {
		t.Errorf("歌单名称错误: got %s, want 通勤", playlist.Name)
	}
	if playlist.SongCount != 0 {
		t.Errorf("歌曲数量错误: got %d, want 0", playlist.SongCount)
	}

	playlists, err := playlistRepo.ListByIdentity(identity.ID)
	if err != nil {
		t.Fatalf("查询歌单列表失败: %v", err)
	}
	if len(playlists) != 1 {
		t.Errorf("歌单数量错误: got %d, want 1", len(playlists))
	}
}

func TestPlaylistRepo_Update(t *testing.T) {
	database := newTestDB(t)
	identityRepo := NewIdentityRepo(database)
	playlistRepo := NewPlaylistRepo(database)

	identity, _ := identityRepo.Create("爸爸", "#6366f1", 0)
	playlist, _ := playlistRepo.Create(identity.ID, "通勤", 0)

	newName := "通勤-改"
	updated, err := playlistRepo.Update(playlist.ID, &newName, nil, nil)
	if err != nil {
		t.Fatalf("更新歌单失败: %v", err)
	}
	if updated.Name != "通勤-改" {
		t.Errorf("歌单名称未更新: got %s", updated.Name)
	}
}

func TestPlaylistRepo_Delete(t *testing.T) {
	database := newTestDB(t)
	identityRepo := NewIdentityRepo(database)
	playlistRepo := NewPlaylistRepo(database)

	identity, _ := identityRepo.Create("爸爸", "#6366f1", 0)
	playlist, _ := playlistRepo.Create(identity.ID, "通勤", 0)

	if err := playlistRepo.Delete(playlist.ID); err != nil {
		t.Fatalf("删除歌单失败: %v", err)
	}

	remaining, _ := playlistRepo.ListByIdentity(identity.ID)
	if len(remaining) != 0 {
		t.Errorf("剩余歌单数量错误: got %d, want 0", len(remaining))
	}
}

func TestPlaylistRepo_CountByIdentity(t *testing.T) {
	database := newTestDB(t)
	identityRepo := NewIdentityRepo(database)
	playlistRepo := NewPlaylistRepo(database)

	identity, _ := identityRepo.Create("爸爸", "#6366f1", 0)

	count, err := playlistRepo.CountByIdentity(identity.ID)
	if err != nil {
		t.Fatalf("统计失败: %v", err)
	}
	if count != 0 {
		t.Errorf("初始歌单数错误: got %d, want 0", count)
	}

	if _, err := playlistRepo.Create(identity.ID, "通勤", 0); err != nil {
		t.Fatalf("创建歌单失败: %v", err)
	}

	count, _ = playlistRepo.CountByIdentity(identity.ID)
	if count != 1 {
		t.Errorf("创建后歌单数错误: got %d, want 1", count)
	}
}

func TestPlaylistRepo_Limit(t *testing.T) {
	database := newTestDB(t)
	identityRepo := NewIdentityRepo(database)
	playlistRepo := NewPlaylistRepo(database)

	identity, _ := identityRepo.Create("爸爸", "#6366f1", 0)

	for i := 0; i < 50; i++ {
		if _, err := playlistRepo.Create(identity.ID, "歌单", i); err != nil {
			t.Fatalf("创建歌单失败: %v", err)
		}
	}

	count, _ := playlistRepo.CountByIdentity(identity.ID)
	if count != 50 {
		t.Errorf("歌单数量错误: got %d, want 50", count)
	}
}
