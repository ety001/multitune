package repository

import (
	"testing"
)

func TestSongRepo_UpsertAndGet(t *testing.T) {
	database := newTestDB(t)
	repo := NewSongRepo(database)

	result, err := repo.Upsert("/app/media/home/song.mp3", "home", "Test Song", "Artist", "Album", 180)
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}
	if !result.IsNew {
		t.Error("首次 Upsert 应为新增")
	}
	if result.Song.Title != "Test Song" {
		t.Errorf("title = %s, want Test Song", result.Song.Title)
	}

	// 再次 Upsert 同一路径应更新
	result2, err := repo.Upsert("/app/media/home/song.mp3", "home", "Updated Song", "Artist2", "Album2", 200)
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}
	if result2.IsNew {
		t.Error("重复 Upsert 应为更新")
	}
	if result2.Song.Title != "Updated Song" {
		t.Errorf("title = %s, want Updated Song", result2.Song.Title)
	}
}

func TestSongRepo_GetByID_NotFound(t *testing.T) {
	database := newTestDB(t)
	repo := NewSongRepo(database)

	song, err := repo.GetByID("nonexistent")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if song != nil {
		t.Error("不存在的歌曲应返回 nil")
	}
}

func TestSongRepo_List(t *testing.T) {
	database := newTestDB(t)
	repo := NewSongRepo(database)

	if _, err := repo.Upsert("/app/media/home/a.mp3", "home", "Alpha", "Artist", "Album", 100); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Upsert("/app/media/usb/b.mp3", "usb", "Beta", "Artist", "Album", 200); err != nil {
		t.Fatal(err)
	}

	songs, total, err := repo.List("", "", 10, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(songs) != 2 {
		t.Errorf("len(songs) = %d, want 2", len(songs))
	}

	// 按来源过滤
	songs, total, err = repo.List("", "usb", 10, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 1 {
		t.Errorf("filtered total = %d, want 1", total)
	}

	// 按关键词搜索
	songs, total, err = repo.List("Alpha", "", 10, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 1 || songs[0].Title != "Alpha" {
		t.Errorf("search result wrong: total=%d, title=%s", total, songs[0].Title)
	}
}

func TestSongRepo_UpsertPreservesCoverURL(t *testing.T) {
	database := newTestDB(t)
	repo := NewSongRepo(database)

	// 首次插入
	result, err := repo.Upsert("/app/media/home/song.mp3", "home", "Title", "Artist", "Album", 100)
	if err != nil {
		t.Fatal(err)
	}
	songID := result.Song.ID

	// 模拟后续封面提取设置了 cover_url
	_, err = database.Exec(`UPDATE songs SET cover_url = ? WHERE id = ?`, "/covers/abc.jpg", songID)
	if err != nil {
		t.Fatal(err)
	}

	// 再次 Upsert（re-scan），cover_url 应保留
	result2, err := repo.Upsert("/app/media/home/song.mp3", "home", "Title-改", "Artist2", "Album2", 200)
	if err != nil {
		t.Fatal(err)
	}
	if result2.IsNew {
		t.Error("re-scan 应为更新而非新增")
	}
	if result2.Song.CoverURL != "/covers/abc.jpg" {
		t.Errorf("cover_url 应被保留: got %q, want /covers/abc.jpg", result2.Song.CoverURL)
	}
	if result2.Song.Title != "Title-改" {
		t.Errorf("title 应更新: got %s, want Title-改", result2.Song.Title)
	}
}

func TestSongRepo_ListEmpty(t *testing.T) {
	database := newTestDB(t)
	repo := NewSongRepo(database)

	// 空列表应返回空 slice 而非 nil（JSON [] 而非 null）
	songs, total, err := repo.List("", "", 10, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if songs == nil {
		t.Error("空列表应返回空 slice 而非 nil")
	}
	if len(songs) != 0 {
		t.Errorf("len(songs) = %d, want 0", len(songs))
	}
}
