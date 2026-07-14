package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHandler_StreamSong(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	// 创建真实音频文件并直接入库
	sourceDir := filepath.Join(h.cfg.MediaRoot, "home")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	songPath := filepath.Join(sourceDir, "test.mp3")
	content := []byte("dummy audio content for streaming test")
	if err := os.WriteFile(songPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	result, err := h.songRepo.Upsert(songPath, "home", "Test", "", "", 100)
	if err != nil {
		t.Fatalf("Upsert song failed: %v", err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/songs/"+result.Song.ID+"/stream", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码错误: got %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "audio/mpeg" {
		t.Errorf("Content-Type = %q, want audio/mpeg", ct)
	}
	if body := w.Body.Bytes(); string(body) != string(content) {
		t.Errorf("响应体错误: got %q, want %q", body, content)
	}
}

func TestHandler_StreamSong_Range(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	sourceDir := filepath.Join(h.cfg.MediaRoot, "home")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	songPath := filepath.Join(sourceDir, "test.mp3")
	content := []byte("0123456789")
	if err := os.WriteFile(songPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	result, err := h.songRepo.Upsert(songPath, "home", "Test", "", "", 100)
	if err != nil {
		t.Fatalf("Upsert song failed: %v", err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/songs/"+result.Song.ID+"/stream", nil)
	req.Header.Set("Range", "bytes=2-5")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusPartialContent {
		t.Fatalf("状态码错误: got %d, want %d", w.Code, http.StatusPartialContent)
	}
	if body := w.Body.String(); body != "2345" {
		t.Errorf("Range 响应体错误: got %q, want 2345", body)
	}
}

func TestHandler_StreamSong_NotFound(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/songs/nonexistent/stream", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("状态码错误: got %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandler_StreamSong_FileMissing(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	// 直接入库一个指向不存在文件的路径
	song, _ := h.songRepo.Upsert(filepath.Join(h.cfg.MediaRoot, "home", "missing.mp3"), "home", "Missing", "", "", 100)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/songs/"+song.Song.ID+"/stream", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("状态码错误: got %d, want %d", w.Code, http.StatusNotFound)
	}
}
