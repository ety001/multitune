package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ety001/multitune/internal/model"
)

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
