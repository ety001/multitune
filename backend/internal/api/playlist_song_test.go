package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ety001/multitune/internal/model"
)

func TestHandler_AddSongsToPlaylist(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	identity := createIdentityForTest(t, r, "爸爸")
	playlist := createPlaylistForTest(t, r, identity.ID, "通勤")
	song := createSongForTest(t, h, r, "home", "song.mp3")

	body := map[string]interface{}{
		"song_ids": []string{song.ID},
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/playlists/"+playlist.ID+"/songs", bytes.NewBuffer(jsonBody))
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

func TestHandler_RemoveSongFromPlaylist(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	identity := createIdentityForTest(t, r, "爸爸")
	playlist := createPlaylistForTest(t, r, identity.ID, "通勤")
	song := createSongForTest(t, h, r, "home", "song.mp3")

	// 先添加
	body := map[string]interface{}{
		"song_ids": []string{song.ID},
	}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/playlists/"+playlist.ID+"/songs", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// 再移除
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/api/playlists/"+playlist.ID+"/songs/"+song.ID, nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码错误: got %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestHandler_UpdatePlaylistSongOrder(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	identity := createIdentityForTest(t, r, "爸爸")
	playlist := createPlaylistForTest(t, r, identity.ID, "通勤")
	song1 := createSongForTest(t, h, r, "home", "song1.mp3")
	song2 := createSongForTest(t, h, r, "home", "song2.mp3")

	// 添加两首歌
	body := map[string]interface{}{
		"song_ids": []string{song1.ID, song2.ID},
	}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/playlists/"+playlist.ID+"/songs", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// 调整顺序
	body = map[string]interface{}{
		"song_ids": []string{song2.ID, song1.ID},
	}
	jsonBody, _ = json.Marshal(body)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/api/playlists/"+playlist.ID+"/songs/order", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码错误: got %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestHandler_GetPlaylist_WithSongs(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	identity := createIdentityForTest(t, r, "爸爸")
	playlist := createPlaylistForTest(t, r, identity.ID, "通勤")
	song := createSongForTest(t, h, r, "home", "song.mp3")

	// 添加歌曲
	body := map[string]interface{}{
		"song_ids": []string{song.ID},
	}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/playlists/"+playlist.ID+"/songs", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// 获取歌单详情
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/playlists/"+playlist.ID, nil)
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

// createSongForTest 测试辅助：扫描一首歌曲并返回
func createSongForTest(t *testing.T, h *Handler, r http.Handler, source, filename string) *model.Song {
	t.Helper()
	sourceDir := filepath.Join(h.cfg.MediaRoot, source)
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	songPath := filepath.Join(sourceDir, filename)
	if err := os.WriteFile(songPath, []byte("dummy audio"), 0644); err != nil {
		t.Fatal(err)
	}

	body := map[string]interface{}{"path": songPath}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/scan", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	var resp model.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析扫描响应失败: %v", err)
	}
	data, _ := json.Marshal(resp.Data)
	var result struct {
		Songs []model.Song `json:"songs"`
	}
	_ = json.Unmarshal(data, &result)
	if len(result.Songs) == 0 {
		t.Fatal("扫描未返回歌曲")
	}
	return &result.Songs[0]
}
