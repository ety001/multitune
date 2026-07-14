package repository

import (
	"testing"
)

func TestPlaybackRepo_SaveAndGet(t *testing.T) {
	database := newTestDB(t)
	repo := NewPlaybackRepo(database)
	identityRepo := NewIdentityRepo(database)
	playlistRepo := NewPlaylistRepo(database)
	songRepo := NewSongRepo(database)

	identity, _ := identityRepo.Create("爸爸", "#6366f1", 0)
	playlist, _ := playlistRepo.Create(identity.ID, "通勤", 0)
	song1, _ := songRepo.Upsert("/app/media/home/a.mp3", "home", "A", "", "", 100)
	song2, _ := songRepo.Upsert("/app/media/home/b.mp3", "home", "B", "", "", 200)

	state, err := repo.Save(identity.ID, playlist.ID, song1.Song.ID, 125, "order")
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if state.Position != 125 {
		t.Errorf("position = %d, want 125", state.Position)
	}
	if state.Mode != "order" {
		t.Errorf("mode = %s, want order", state.Mode)
	}

	got, err := repo.GetByIdentity(identity.ID)
	if err != nil {
		t.Fatalf("GetByIdentity failed: %v", err)
	}
	if got.SongID != song1.Song.ID {
		t.Errorf("song_id = %s, want %s", got.SongID, song1.Song.ID)
	}

	// 更新
	state2, err := repo.Save(identity.ID, playlist.ID, song2.Song.ID, 60, "random")
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if state2.SongID != song2.Song.ID || state2.Mode != "random" {
		t.Errorf("update failed: %+v", state2)
	}
}

func TestPlaybackRepo_SongProgress(t *testing.T) {
	database := newTestDB(t)
	repo := NewPlaybackRepo(database)
	identityRepo := NewIdentityRepo(database)
	songRepo := NewSongRepo(database)

	identity, _ := identityRepo.Create("爸爸", "#6366f1", 0)
	song, _ := songRepo.Upsert("/app/media/home/a.mp3", "home", "A", "", "", 100)

	if err := repo.SaveSongProgress(identity.ID, song.Song.ID, 88); err != nil {
		t.Fatalf("SaveSongProgress failed: %v", err)
	}

	pos, err := repo.GetSongProgress(identity.ID, song.Song.ID)
	if err != nil {
		t.Fatalf("GetSongProgress failed: %v", err)
	}
	if pos != 88 {
		t.Errorf("position = %d, want 88", pos)
	}

	// 更新
	if err := repo.SaveSongProgress(identity.ID, song.Song.ID, 120); err != nil {
		t.Fatalf("SaveSongProgress failed: %v", err)
	}
	pos, _ = repo.GetSongProgress(identity.ID, song.Song.ID)
	if pos != 120 {
		t.Errorf("position = %d, want 120", pos)
	}
}
