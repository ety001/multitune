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

func TestValidateMediaPath(t *testing.T) {
	mediaRoot := t.TempDir()
	subDir := filepath.Join(mediaRoot, "home", "music")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	// 合法路径
	if err := ValidateMediaPath(mediaRoot, subDir); err != nil {
		t.Errorf("合法路径校验失败: %v", err)
	}

	// 目录遍历
	if err := ValidateMediaPath(mediaRoot, filepath.Join(mediaRoot, "..", "etc")); err == nil {
		t.Error("目录遍历应被阻止")
	}

	// 外部路径
	if err := ValidateMediaPath(mediaRoot, "/etc/passwd"); err == nil {
		t.Error("外部路径应被阻止")
	}
}

func TestListSources(t *testing.T) {
	mediaRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(mediaRoot, "home"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(mediaRoot, "usb"), 0755); err != nil {
		t.Fatal(err)
	}

	sources, err := ListSources(mediaRoot)
	if err != nil {
		t.Fatalf("ListSources failed: %v", err)
	}
	if len(sources) != 2 {
		t.Errorf("source count = %d, want 2", len(sources))
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
		if item["name"] == "song.mp3" {
			if isAudio, ok := item["is_audio"].(bool); !ok || !isAudio {
				t.Error("song.mp3 should be marked as audio")
			}
			foundAudio = true
		}
	}
	if !foundAudio {
		t.Error("audio file not found in listing")
	}
}
