package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ety001/multitune/internal/model"
)

// addSongToPlaylistForTest 将歌曲加入歌单（供播放状态测试使用）
func addSongToPlaylistForTest(t *testing.T, r http.Handler, playlistID, songID string) {
	t.Helper()
	body := map[string]interface{}{
		"song_ids": []string{songID},
	}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/playlists/"+playlistID+"/songs", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("添加歌曲到歌单失败: %d, body: %s", w.Code, w.Body.String())
	}
}

func TestHandler_GetPlaybackState(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	identity := createIdentityForTest(t, r, "爸爸")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/playback/"+identity.ID, nil)
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

func TestHandler_SavePlaybackState(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	identity := createIdentityForTest(t, r, "爸爸")
	playlist := createPlaylistForTest(t, r, identity.ID, "通勤")
	song := createSongForTest(t, h, r, "home", "song.mp3")
	addSongToPlaylistForTest(t, r, playlist.ID, song.ID)

	body := map[string]interface{}{
		"playlist_id": playlist.ID,
		"song_id":     song.ID,
		"position":    125,
		"mode":        "random",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/playback/"+identity.ID, bytes.NewBuffer(jsonBody))
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

	data, _ := json.Marshal(resp.Data)
	var state model.PlaybackState
	_ = json.Unmarshal(data, &state)
	if state.Mode != "random" || state.Position != 125 {
		t.Errorf("保存状态错误: %+v", state)
	}
}

func TestHandler_SavePlaybackState_InvalidMode(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	identity := createIdentityForTest(t, r, "爸爸")

	body := map[string]interface{}{
		"mode": "invalid",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/playback/"+identity.ID, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("状态码错误: got %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandler_GetPlaybackState_InvalidIdentity(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/playback/nonexistent", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("状态码错误: got %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandler_GetPlaybackState_DefaultState(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	identity := createIdentityForTest(t, r, "爸爸")

	// 无保存状态时获取默认空状态
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/playback/"+identity.ID, nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码错误: got %d", w.Code)
	}

	var resp model.APIResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data, _ := json.Marshal(resp.Data)
	var state model.PlaybackState
	_ = json.Unmarshal(data, &state)

	if state.Mode != "order" {
		t.Errorf("默认 mode = %s, want order", state.Mode)
	}
	if state.Position != 0 {
		t.Errorf("默认 position = %d, want 0", state.Position)
	}
}

func TestHandler_SavePlaybackState_InvalidPlaylist(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	identity := createIdentityForTest(t, r, "爸爸")

	body := map[string]interface{}{
		"playlist_id": "nonexistent-playlist",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/playback/"+identity.ID, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("状态码错误: got %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp model.APIResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != ErrCodePlaylistNotFound {
		t.Errorf("错误码错误: got %d, want %d", resp.Code, ErrCodePlaylistNotFound)
	}
}

func TestHandler_SavePlaybackState_InvalidSong(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	identity := createIdentityForTest(t, r, "爸爸")

	body := map[string]interface{}{
		"song_id": "nonexistent-song",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/playback/"+identity.ID, bytes.NewBuffer(jsonBody))
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

func TestHandler_SavePlaybackState_PartialUpdate(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	identity := createIdentityForTest(t, r, "爸爸")
	playlist := createPlaylistForTest(t, r, identity.ID, "通勤")
	song := createSongForTest(t, h, r, "home", "song.mp3")
	addSongToPlaylistForTest(t, r, playlist.ID, song.ID)

	// 先完整保存一次
	body := map[string]interface{}{
		"playlist_id": playlist.ID,
		"song_id":     song.ID,
		"position":    100,
		"mode":        "order",
	}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/playback/"+identity.ID, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// 只更新 position，其他字段应保持不变
	body = map[string]interface{}{
		"position": 200,
	}
	jsonBody, _ = json.Marshal(body)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/playback/"+identity.ID, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码错误: got %d, body: %s", w.Code, w.Body.String())
	}

	var resp model.APIResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data, _ := json.Marshal(resp.Data)
	var state model.PlaybackState
	_ = json.Unmarshal(data, &state)

	if state.Position != 200 {
		t.Errorf("position = %d, want 200", state.Position)
	}
	if state.Mode != "order" {
		t.Errorf("mode = %s, want order（应保持不变）", state.Mode)
	}
	if state.PlaylistID != playlist.ID {
		t.Errorf("playlist_id = %s, want %s（应保持不变）", state.PlaylistID, playlist.ID)
	}
}

func TestHandler_SavePlaybackState_SongProgressSaved(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	identity := createIdentityForTest(t, r, "爸爸")
	song := createSongForTest(t, h, r, "home", "song.mp3")

	// 保存播放状态（含 song_id + position）
	body := map[string]interface{}{
		"song_id":  song.ID,
		"position": 88,
		"mode":     "random",
	}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/playback/"+identity.ID, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// 验证 song_progress 表有数据
	pos, err := h.playbackRepo.GetSongProgress(identity.ID, song.ID)
	if err != nil {
		t.Fatalf("查询单曲进度失败: %v", err)
	}
	if pos != 88 {
		t.Errorf("单曲进度 = %d, want 88", pos)
	}
}

func TestHandler_SavePlaybackState_NegativePosition(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	identity := createIdentityForTest(t, r, "爸爸")

	body := map[string]interface{}{
		"position": -10,
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/playback/"+identity.ID, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("状态码错误: got %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandler_SavePlaybackState_SongNotInPlaylist(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	identity := createIdentityForTest(t, r, "爸爸")
	playlist := createPlaylistForTest(t, r, identity.ID, "通勤")
	song := createSongForTest(t, h, r, "home", "song.mp3")
	// 注意：song 未加入 playlist

	body := map[string]interface{}{
		"playlist_id": playlist.ID,
		"song_id":     song.ID,
		"position":    10,
		"mode":        "order",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/playback/"+identity.ID, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("状态码错误: got %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}

	var resp model.APIResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != ErrCodeSongNotInPlaylist {
		t.Errorf("错误码错误: got %d, want %d", resp.Code, ErrCodeSongNotInPlaylist)
	}
}

func TestHandler_GetPlaylistProgress_NotFound(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/playlists/nonexistent/progress", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("状态码错误: got %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandler_GetPlaylistProgress_DefaultEmpty(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	identity := createIdentityForTest(t, r, "爸爸")
	playlist := createPlaylistForTest(t, r, identity.ID, "通勤")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/playlists/"+playlist.ID+"/progress", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码错误: got %d, body: %s", w.Code, w.Body.String())
	}

	var resp model.APIResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data, _ := json.Marshal(resp.Data)
	var state model.PlaylistState
	_ = json.Unmarshal(data, &state)

	if state.PlaylistID != playlist.ID {
		t.Errorf("playlist_id = %s, want %s", state.PlaylistID, playlist.ID)
	}
	if state.SongID != "" {
		t.Errorf("默认 song_id = %s, want 空", state.SongID)
	}
	if state.Position != 0 {
		t.Errorf("默认 position = %d, want 0", state.Position)
	}
}

func TestHandler_GetPlaylistProgress_AfterSave(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	identity := createIdentityForTest(t, r, "爸爸")
	playlist := createPlaylistForTest(t, r, identity.ID, "通勤")
	song := createSongForTest(t, h, r, "home", "song.mp3")
	addSongToPlaylistForTest(t, r, playlist.ID, song.ID)

	// 保存播放状态
	body := map[string]interface{}{
		"playlist_id": playlist.ID,
		"song_id":     song.ID,
		"position":    125,
		"mode":        "order",
	}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/playback/"+identity.ID, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("保存状态码错误: got %d, body: %s", w.Code, w.Body.String())
	}

	// 读取歌单记忆点
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/playlists/"+playlist.ID+"/progress", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码错误: got %d, body: %s", w.Code, w.Body.String())
	}

	var resp model.APIResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data, _ := json.Marshal(resp.Data)
	var state model.PlaylistState
	_ = json.Unmarshal(data, &state)

	if state.SongID != song.ID {
		t.Errorf("song_id = %s, want %s", state.SongID, song.ID)
	}
	if state.Position != 125 {
		t.Errorf("position = %d, want 125", state.Position)
	}

	// 只发 position 部分更新，歌单记忆点应保持正确的歌并更新位置
	body = map[string]interface{}{
		"position": 200,
	}
	jsonBody, _ = json.Marshal(body)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/playback/"+identity.ID, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("部分更新状态码错误: got %d, body: %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/playlists/"+playlist.ID+"/progress", nil)
	r.ServeHTTP(w, req)

	var resp2 model.APIResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp2)
	data, _ = json.Marshal(resp2.Data)
	var state2 model.PlaylistState
	_ = json.Unmarshal(data, &state2)

	if state2.SongID != song.ID {
		t.Errorf("部分更新后 song_id = %s, want %s", state2.SongID, song.ID)
	}
	if state2.Position != 200 {
		t.Errorf("部分更新后 position = %d, want 200", state2.Position)
	}
}
