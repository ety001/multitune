package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ety001/multitune/internal/model"
)

func TestHandler_ListStorageSources(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	// 创建测试媒体源目录
	if err := os.MkdirAll(filepath.Join(t.TempDir(), "home"), 0755); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/fs/sources", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码错误: got %d, want %d", w.Code, http.StatusOK)
	}

	var resp model.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Code != 0 {
		t.Errorf("响应码错误: got %d, want 0", resp.Code)
	}
}

func TestHandler_ListDirectory(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	musicDir := filepath.Join(t.TempDir(), "home", "music")
	if err := os.MkdirAll(musicDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(musicDir, "song.mp3"), []byte("dummy"), 0644); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/fs/list?path="+musicDir, nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码错误: got %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp model.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Code != 0 {
		t.Errorf("响应码错误: got %d, want 0", resp.Code)
	}
}

func TestHandler_ListDirectory_InvalidPath(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/fs/list?path=/etc/passwd", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("状态码错误: got %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandler_ScanSongs(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	musicDir := filepath.Join(t.TempDir(), "home", "music")
	if err := os.MkdirAll(musicDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(musicDir, "song.mp3"), []byte("dummy"), 0644); err != nil {
		t.Fatal(err)
	}

	body := map[string]interface{}{"path": musicDir}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/scan", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码错误: got %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp model.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Code != 0 {
		t.Errorf("响应码错误: got %d, want 0", resp.Code)
	}
}

func TestHandler_ListSongs(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	// 先扫描一首歌曲
	musicDir := filepath.Join(t.TempDir(), "home", "music")
	if err := os.MkdirAll(musicDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(musicDir, "song.mp3"), []byte("dummy"), 0644); err != nil {
		t.Fatal(err)
	}

	body := map[string]interface{}{"path": musicDir}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/scan", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// 查询歌曲列表
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/songs", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码错误: got %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp model.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Code != 0 {
		t.Errorf("响应码错误: got %d, want 0", resp.Code)
	}
}

func TestHandler_GetSong_NotFound(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/songs/nonexistent", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("状态码错误: got %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandler_BatchGetSongs(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	// 直接用 repo 建测试歌曲（不走扫描）
	song1, _ := h.songRepo.Upsert("/a.mp3", "home", "A", "", "", 100)
	song2, _ := h.songRepo.Upsert("/b.mp3", "home", "B", "", "", 200)

	// 成功：批量查询存在的歌
	body, _ := json.Marshal(map[string]interface{}{
		"ids": []string{song1.Song.ID, song2.Song.ID},
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/songs/batch", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码错误: got %d, body: %s", w.Code, w.Body.String())
	}
	var resp model.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Code != 0 {
		t.Errorf("响应码错误: got %d, want 0", resp.Code)
	}

	// 空数组：应返回 200 + 空 songs（不是 null）
	body, _ = json.Marshal(map[string]interface{}{"ids": []string{}})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/songs/batch", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("空数组状态码错误: got %d", w.Code)
	}
	respRaw := w.Body.String()
	if !strings.Contains(respRaw, `"songs":[]`) {
		t.Errorf("空数组应返回 songs:[]，got: %s", respRaw)
	}

	// 超过 100 个：应返回 400
	bigIDs := make([]string, 101)
	for i := range bigIDs {
		bigIDs[i] = "x"
	}
	body, _ = json.Marshal(map[string]interface{}{"ids": bigIDs})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/songs/batch", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("超限应返回 400，got %d", w.Code)
	}

	// 含不存在的 id：不报错，只返回存在的
	body, _ = json.Marshal(map[string]interface{}{
		"ids": []string{song1.Song.ID, "nonexistent"},
	})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/songs/batch", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("含不存在 id 应返回 200，got %d", w.Code)
	}
}
