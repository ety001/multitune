package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ety001/multitune/internal/config"
	"github.com/ety001/multitune/internal/db"
	"github.com/ety001/multitune/internal/repository"
)

func newTestScanner(t *testing.T) (*Scanner, string) {
	t.Helper()
	mediaRoot := t.TempDir()
	cfg := &config.Config{
		DataPath:                t.TempDir(),
		DatabaseName:            "test.db",
		MaxIdentities:           20,
		MaxPlaylistsPerIdentity: 50,
		MaxSongsPerPlaylist:     1000,
		ScanFormats:             []string{"mp3", "flac", "m4a", "aac", "ogg", "wav"},
	}
	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	t.Cleanup(func() {
		_ = database.Close()
	})
	repo := repository.NewSongRepo(database)
	return New(mediaRoot, repo, cfg.ScanFormats), mediaRoot
}

func TestScanner_ScanFile(t *testing.T) {
	scanner, mediaRoot := newTestScanner(t)
	sourceDir := filepath.Join(mediaRoot, "home", "music")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	songPath := filepath.Join(sourceDir, "test.mp3")
	if err := os.WriteFile(songPath, []byte("dummy audio data"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := scanner.ScanPath(songPath)
	if err != nil {
		t.Fatalf("ScanPath failed: %v", err)
	}
	if result.Scanned != 1 {
		t.Errorf("scanned = %d, want 1", result.Scanned)
	}
	if result.Added != 1 {
		t.Errorf("added = %d, want 1", result.Added)
	}
	if len(result.Songs) != 1 {
		t.Fatalf("songs count = %d, want 1", len(result.Songs))
	}
	if result.Songs[0].Source != "home" {
		t.Errorf("source = %s, want home", result.Songs[0].Source)
	}
	if result.Songs[0].Title != "test" {
		// ffprobe 不可用时使用文件名作为标题
		t.Errorf("title = %s, want test (fallback title)", result.Songs[0].Title)
	}
}

func TestScanner_ScanDirectory(t *testing.T) {
	scanner, mediaRoot := newTestScanner(t)
	sourceDir := filepath.Join(mediaRoot, "usb", "songs")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "a.mp3"), []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "b.flac"), []byte("b"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "readme.txt"), []byte("txt"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := scanner.ScanPath(sourceDir)
	if err != nil {
		t.Fatalf("ScanPath failed: %v", err)
	}
	if result.Scanned != 2 {
		t.Errorf("scanned = %d, want 2", result.Scanned)
	}
	if result.Added != 2 {
		t.Errorf("added = %d, want 2", result.Added)
	}
}

func TestScanner_InferSource(t *testing.T) {
	scanner, mediaRoot := newTestScanner(t)

	sourceDir := filepath.Join(mediaRoot, "usb", "songs")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	songPath := filepath.Join(sourceDir, "song.mp3")
	if err := os.WriteFile(songPath, []byte("dummy"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := scanner.ScanPath(songPath)
	if err != nil {
		t.Fatalf("ScanPath failed: %v", err)
	}
	if result.Songs[0].Source != "usb" {
		t.Errorf("source = %s, want usb", result.Songs[0].Source)
	}
}
