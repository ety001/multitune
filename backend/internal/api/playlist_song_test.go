package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ety001/multitune/internal/config"
	"github.com/ety001/multitune/internal/db"
	"github.com/ety001/multitune/internal/model"
	"github.com/gin-gonic/gin"
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

func TestHandler_AddSongsToPlaylist_SongNotFound(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	identity := createIdentityForTest(t, r, "爸爸")
	playlist := createPlaylistForTest(t, r, identity.ID, "通勤")

	body := map[string]interface{}{
		"song_ids": []string{"nonexistent-song-id"},
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/playlists/"+playlist.ID+"/songs", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("状态码错误: got %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp model.APIResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != ErrCodeSongNotFound {
		t.Errorf("错误码错误: got %d, want %d", resp.Code, ErrCodeSongNotFound)
	}
}

func TestHandler_AddSongsToPlaylist_ExceedLimit(t *testing.T) {
	cfg := &config.Config{
		DataPath:                t.TempDir(),
		DatabaseName:            "test.db",
		MediaRoot:               t.TempDir(),
		MaxIdentities:           20,
		MaxPlaylistsPerIdentity: 50,
		MaxSongsPerPlaylist:     1,
		GINMode:                 gin.TestMode,
		StaticPath:              "/nonexistent",
	}
	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	gin.DefaultWriter = os.NewFile(0, os.DevNull)
	h := NewHandler(cfg, database)
	r := h.SetupRouter()

	identity := createIdentityForTest(t, r, "爸爸")
	playlist := createPlaylistForTest(t, r, identity.ID, "通勤")
	song1 := createSongForTest(t, h, r, "home", "song1.mp3")
	song2 := createSongForTest(t, h, r, "home", "song2.mp3")

	// 添加 2 首到上限为 1 的歌单
	body := map[string]interface{}{
		"song_ids": []string{song1.ID, song2.ID},
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/playlists/"+playlist.ID+"/songs", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("状态码错误: got %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp model.APIResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != ErrCodePlaylistSongLimit {
		t.Errorf("错误码错误: got %d, want %d", resp.Code, ErrCodePlaylistSongLimit)
	}
}

func TestHandler_UpdatePlaylistSongOrder_SongNotInPlaylist(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	identity := createIdentityForTest(t, r, "爸爸")
	playlist := createPlaylistForTest(t, r, identity.ID, "通勤")
	song1 := createSongForTest(t, h, r, "home", "song1.mp3")
	createSongForTest(t, h, r, "home", "song2.mp3") // not added to playlist

	// 只添加 song1
	body := map[string]interface{}{
		"song_ids": []string{song1.ID},
	}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/playlists/"+playlist.ID+"/songs", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// 尝试排序包含不在歌单中的歌曲
	body = map[string]interface{}{
		"song_ids": []string{song1.ID, "nonexistent-in-playlist"},
	}
	jsonBody, _ = json.Marshal(body)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/api/playlists/"+playlist.ID+"/songs/order", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("状态码错误: got %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp model.APIResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != ErrCodeSongNotInPlaylist {
		t.Errorf("错误码错误: got %d, want %d", resp.Code, ErrCodeSongNotInPlaylist)
	}
}

func TestHandler_GetPlaylist_Pagination(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	identity := createIdentityForTest(t, r, "爸爸")
	playlist := createPlaylistForTest(t, r, identity.ID, "通勤")

	// 创建 3 首歌并添加
	var songIDs []string
	for i := 0; i < 3; i++ {
		song := createSongForTest(t, h, r, "home", "song"+string(rune('1'+i))+".mp3")
		songIDs = append(songIDs, song.ID)
	}

	body := map[string]interface{}{"song_ids": songIDs}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/playlists/"+playlist.ID+"/songs", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// 请求 limit=2 offset=0
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/playlists/"+playlist.ID+"?limit=2&offset=0", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码错误: got %d, body: %s", w.Code, w.Body.String())
	}

	var resp model.APIResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data, _ := json.Marshal(resp.Data)
	var detail struct {
		SongCount int          `json:"song_count"`
		Songs     []model.Song `json:"songs"`
	}
	_ = json.Unmarshal(data, &detail)

	if detail.SongCount != 3 {
		t.Errorf("song_count = %d, want 3", detail.SongCount)
	}
	if len(detail.Songs) != 2 {
		t.Errorf("songs len = %d, want 2 (limit)", len(detail.Songs))
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
