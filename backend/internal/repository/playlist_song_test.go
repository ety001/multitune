package repository

import (
	"testing"
)

func TestPlaylistRepo_AddSongs(t *testing.T) {
	database := newTestDB(t)
	identityRepo := NewIdentityRepo(database)
	playlistRepo := NewPlaylistRepo(database)
	songRepo := NewSongRepo(database)

	identity, _ := identityRepo.Create("爸爸", "#6366f1", 0)
	playlist, _ := playlistRepo.Create(identity.ID, "通勤", 0)

	song1, _ := songRepo.Upsert("/app/media/home/a.mp3", "home", "A", "", "", 100)
	song2, _ := songRepo.Upsert("/app/media/home/b.mp3", "home", "B", "", "", 200)

	added, err := playlistRepo.AddSongs(playlist.ID, []string{song1.Song.ID, song2.Song.ID})
	if err != nil {
		t.Fatalf("AddSongs failed: %v", err)
	}
	if added != 2 {
		t.Errorf("added = %d, want 2", added)
	}

	// 重复添加不应报错，且实际新增数为 0
	added, err = playlistRepo.AddSongs(playlist.ID, []string{song1.Song.ID})
	if err != nil {
		t.Fatalf("AddSongs failed: %v", err)
	}
	if added != 0 {
		t.Errorf("重复添加实际新增数应为 0，added = %d", added)
	}
}

func TestPlaylistRepo_RemoveSong(t *testing.T) {
	database := newTestDB(t)
	identityRepo := NewIdentityRepo(database)
	playlistRepo := NewPlaylistRepo(database)
	songRepo := NewSongRepo(database)

	identity, _ := identityRepo.Create("爸爸", "#6366f1", 0)
	playlist, _ := playlistRepo.Create(identity.ID, "通勤", 0)
	song, _ := songRepo.Upsert("/app/media/home/a.mp3", "home", "A", "", "", 100)
	playlistRepo.AddSongs(playlist.ID, []string{song.Song.ID})

	if err := playlistRepo.RemoveSong(playlist.ID, song.Song.ID); err != nil {
		t.Fatalf("RemoveSong failed: %v", err)
	}

	count, _ := playlistRepo.CountSongs(playlist.ID)
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
}

func TestPlaylistRepo_UpdateSongOrder(t *testing.T) {
	database := newTestDB(t)
	identityRepo := NewIdentityRepo(database)
	playlistRepo := NewPlaylistRepo(database)
	songRepo := NewSongRepo(database)

	identity, _ := identityRepo.Create("爸爸", "#6366f1", 0)
	playlist, _ := playlistRepo.Create(identity.ID, "通勤", 0)
	song1, _ := songRepo.Upsert("/app/media/home/a.mp3", "home", "A", "", "", 100)
	song2, _ := songRepo.Upsert("/app/media/home/b.mp3", "home", "B", "", "", 200)
	playlistRepo.AddSongs(playlist.ID, []string{song1.Song.ID, song2.Song.ID})

	// 调整顺序：b 在前，a 在后
	err := playlistRepo.UpdateSongOrder(playlist.ID, []string{song2.Song.ID, song1.Song.ID})
	if err != nil {
		t.Fatalf("UpdateSongOrder failed: %v", err)
	}

	songs, _, _ := playlistRepo.ListSongs(playlist.ID, 10, 0)
	if len(songs) != 2 {
		t.Fatalf("songs count = %d, want 2", len(songs))
	}
	if songs[0].ID != song2.Song.ID {
		t.Errorf("first song = %s, want %s", songs[0].ID, song2.Song.ID)
	}
}

func TestPlaylistRepo_ListSongs(t *testing.T) {
	database := newTestDB(t)
	identityRepo := NewIdentityRepo(database)
	playlistRepo := NewPlaylistRepo(database)
	songRepo := NewSongRepo(database)

	identity, _ := identityRepo.Create("爸爸", "#6366f1", 0)
	playlist, _ := playlistRepo.Create(identity.ID, "通勤", 0)
	song, _ := songRepo.Upsert("/app/media/home/a.mp3", "home", "A", "", "", 100)
	playlistRepo.AddSongs(playlist.ID, []string{song.Song.ID})

	songs, total, err := playlistRepo.ListSongs(playlist.ID, 10, 0)
	if err != nil {
		t.Fatalf("ListSongs failed: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(songs) != 1 {
		t.Errorf("len(songs) = %d, want 1", len(songs))
	}
}

func TestPlaylistRepo_ListSongIDs(t *testing.T) {
	database := newTestDB(t)
	identityRepo := NewIdentityRepo(database)
	playlistRepo := NewPlaylistRepo(database)
	songRepo := NewSongRepo(database)

	identity, _ := identityRepo.Create("爸爸", "#6366f1", 0)
	playlist, _ := playlistRepo.Create(identity.ID, "通勤", 0)
	song1, _ := songRepo.Upsert("/a.mp3", "home", "A", "", "", 100)
	song2, _ := songRepo.Upsert("/b.mp3", "home", "B", "", "", 200)
	song3, _ := songRepo.Upsert("/c.mp3", "home", "C", "", "", 300)

	// 按顺序添加：song3, song1, song2（验证按 sort_order 而非添加顺序）
	playlistRepo.AddSongs(playlist.ID, []string{song3.Song.ID})
	playlistRepo.AddSongs(playlist.ID, []string{song1.Song.ID})
	playlistRepo.AddSongs(playlist.ID, []string{song2.Song.ID})

	ids, err := playlistRepo.ListSongIDs(playlist.ID)
	if err != nil {
		t.Fatalf("ListSongIDs failed: %v", err)
	}
	if len(ids) != 3 {
		t.Fatalf("len(ids) = %d, want 3", len(ids))
	}

	// 应与 ListSongs 返回的顺序一致（都是 sort_order ASC, created_at ASC）
	songs, _, _ := playlistRepo.ListSongs(playlist.ID, 10, 0)
	if len(songs) != len(ids) {
		t.Fatalf("ListSongs 与 ListSongIDs 数量不一致")
	}
	for i := range ids {
		if ids[i] != songs[i].ID {
			t.Errorf("位置 %d 顺序不一致：ListSongIDs=%s, ListSongs=%s", i, ids[i], songs[i].ID)
		}
	}

	// 空歌单应返回空 slice 而非 nil
	empty, _ := playlistRepo.Create(identity.ID, "空歌单", 1)
	ids, err = playlistRepo.ListSongIDs(empty.ID)
	if err != nil {
		t.Fatalf("空歌单 ListSongIDs failed: %v", err)
	}
	if ids == nil {
		t.Error("空歌单应返回空 slice 而非 nil")
	}
	if len(ids) != 0 {
		t.Errorf("空歌单 len = %d, want 0", len(ids))
	}
}
