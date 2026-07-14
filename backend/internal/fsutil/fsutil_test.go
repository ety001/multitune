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

func TestValidateMediaPath_SymlinkEscape(t *testing.T) {
	mediaRoot := t.TempDir()
	targetDir := t.TempDir() // mediaRoot 外部的目标目录

	// 在 mediaRoot 下创建一个子目录
	homeDir := filepath.Join(mediaRoot, "home")
	if err := os.MkdirAll(homeDir, 0755); err != nil {
		t.Fatal(err)
	}

	// 创建一个软链接指向 mediaRoot 外部
	linkPath := filepath.Join(homeDir, "escape")
	if err := os.Symlink(targetDir, linkPath); err != nil {
		t.Fatal(err)
	}

	// 通过软链接访问 mediaRoot 外部应被阻止
	escapePath := filepath.Join(linkPath, "secret.txt")
	if err := ValidateMediaPath(mediaRoot, escapePath); err == nil {
		t.Error("软链接跳出 mediaRoot 应被阻止")
	}
}

func TestValidateMediaPath_NestedSymlink(t *testing.T) {
	mediaRoot := t.TempDir()
	targetDir := t.TempDir()

	// 创建多层目录
	musicDir := filepath.Join(mediaRoot, "home", "music")
	if err := os.MkdirAll(musicDir, 0755); err != nil {
		t.Fatal(err)
	}

	// 在子目录中创建指向外部的软链接
	linkPath := filepath.Join(musicDir, "external")
	if err := os.Symlink(targetDir, linkPath); err != nil {
		t.Fatal(err)
	}

	// 通过嵌套软链接访问外部应被阻止
	escapePath := filepath.Join(linkPath, "file.txt")
	if err := ValidateMediaPath(mediaRoot, escapePath); err == nil {
		t.Error("嵌套软链接跳出应被阻止")
	}

	// 正常的 mediaRoot 内路径仍然合法
	normalPath := filepath.Join(musicDir, "song.mp3")
	if err := ValidateMediaPath(mediaRoot, normalPath); err != nil {
		t.Errorf("mediaRoot 内的正常路径应合法: %v", err)
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
