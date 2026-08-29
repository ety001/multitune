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

	deleted, err := playlistRepo.Delete(playlist.ID)
	if err != nil {
		t.Fatalf("删除歌单失败: %v", err)
	}
	if !deleted {
		t.Error("应返回 deleted=true")
	}

	remaining, _ := playlistRepo.ListByIdentity(identity.ID)
	if len(remaining) != 0 {
		t.Errorf("剩余歌单数量错误: got %d, want 0", len(remaining))
	}

	// 删除不存在的歌单
	deleted2, err := playlistRepo.Delete(playlist.ID)
	if err != nil {
		t.Fatalf("删除不存在歌单应无错误: %v", err)
	}
	if deleted2 {
		t.Error("不存在的歌单应返回 deleted=false")
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

// 播放某歌单会刷新 playlist_states.updated_at，列表应按其逆序（最近播放优先）
func TestPlaylistRepo_ListOrderByLastPlayed(t *testing.T) {
	database := newTestDB(t)
	identityRepo := NewIdentityRepo(database)
	playlistRepo := NewPlaylistRepo(database)
	playbackRepo := NewPlaybackRepo(database)

	identity, err := identityRepo.Create("爸爸", "#6366f1", 0)
	if err != nil {
		t.Fatalf("创建身份失败: %v", err)
	}

	older, _ := playlistRepo.Create(identity.ID, "较早播放", 0)
	newer, _ := playlistRepo.Create(identity.ID, "最近播放", 1)
	never, _ := playlistRepo.Create(identity.ID, "从未播放", 2)

	// 先播 older 再播 newer：newer 的记忆点时间更晚。
	// SaveWithProgress 的时间戳为秒级，同秒内两次保存无法区分先后，
	// 这里直接设定明确的记忆点时间。
	if _, err := playbackRepo.SaveWithProgress(identity.ID, strPtr(older.ID), nil, intPtr(10), nil); err != nil {
		t.Fatalf("保存 older 播放状态失败: %v", err)
	}
	if _, err := playbackRepo.SaveWithProgress(identity.ID, strPtr(newer.ID), nil, intPtr(20), nil); err != nil {
		t.Fatalf("保存 newer 播放状态失败: %v", err)
	}
	if _, err := database.Exec(`UPDATE playlist_states SET updated_at = 1000 WHERE playlist_id = ?`, older.ID); err != nil {
		t.Fatalf("设定 older 记忆点时间失败: %v", err)
	}
	if _, err := database.Exec(`UPDATE playlist_states SET updated_at = 2000 WHERE playlist_id = ?`, newer.ID); err != nil {
		t.Fatalf("设定 newer 记忆点时间失败: %v", err)
	}

	playlists, err := playlistRepo.ListByIdentity(identity.ID)
	if err != nil {
		t.Fatalf("查询歌单列表失败: %v", err)
	}
	if len(playlists) != 3 {
		t.Fatalf("歌单数量错误: got %d, want 3", len(playlists))
	}
	wantOrder := []string{newer.ID, older.ID, never.ID}
	for i, want := range wantOrder {
		if playlists[i].ID != want {
			t.Errorf("排序错误 位置%d: got %s, want %s", i, playlists[i].ID, want)
		}
	}
}

func TestPlaylistRepo_SearchByIdentity(t *testing.T) {
	database := newTestDB(t)
	identityRepo := NewIdentityRepo(database)
	playlistRepo := NewPlaylistRepo(database)

	identity, _ := identityRepo.Create("爸爸", "#6366f1", 0)
	other, _ := identityRepo.Create("妈妈", "#6366f1", 1)

	// 同一身份下：两个含"通勤"，一个不含；另一身份的同名歌单不应出现
	playlistRepo.Create(identity.ID, "通勤快歌", 0)
	playlistRepo.Create(identity.ID, "睡前通勤", 1)
	playlistRepo.Create(identity.ID, "儿歌", 2)
	playlistRepo.Create(other.ID, "别人的通勤", 0)

	playlists, err := playlistRepo.SearchByIdentity(identity.ID, "通勤")
	if err != nil {
		t.Fatalf("搜索歌单失败: %v", err)
	}
	if len(playlists) != 2 {
		t.Fatalf("搜索结果数量错误: got %d, want 2", len(playlists))
	}
	for _, p := range playlists {
		if p.IdentityID != identity.ID {
			t.Errorf("搜索结果包含其他身份的歌单: %s", p.Name)
		}
	}

	// LIKE 通配符应按字面量匹配，% 不匹配任意串
	escaped, err := playlistRepo.SearchByIdentity(identity.ID, "%")
	if err != nil {
		t.Fatalf("搜索歌单失败: %v", err)
	}
	if len(escaped) != 0 {
		t.Errorf("通配符未转义: got %d 条结果, want 0", len(escaped))
	}

	// 无匹配返回空 slice（JSON 序列化为 []，非 null）
	none, err := playlistRepo.SearchByIdentity(identity.ID, "不存在的歌单")
	if err != nil {
		t.Fatalf("搜索歌单失败: %v", err)
	}
	if none == nil || len(none) != 0 {
		t.Errorf("无匹配应返回空 slice: got %v", none)
	}
}
