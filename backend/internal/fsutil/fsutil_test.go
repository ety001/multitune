package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsAudioFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"song.mp3", true},
		{"song.MP3", true},
		{"song.flac", true},
		{"song.m4a", true},
		{"song.txt", false},
		{"song", false},
	}
	for _, tt := range tests {
		if got := IsAudioFile(tt.path); got != tt.want {
			t.Errorf("IsAudioFile(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestListDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "song.mp3"), []byte("dummy"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0755); err != nil {
		t.Fatal(err)
	}

	items, err := ListDirectory(dir)
	if err != nil {
		t.Fatalf("ListDirectory failed: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("item count = %d, want 2", len(items))
	}

	var foundAudio bool
	for _, item := range items {
		if item.Name == "song.mp3" {
			if !item.IsAudio {
				t.Error("song.mp3 should be marked as audio")
			}
			foundAudio = true
		}
	}
	if !foundAudio {
		t.Error("audio file not found in listing")
	}
}
