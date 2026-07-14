package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ety001/multitune/internal/model"
)

func TestHandler_ListPlaylists(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	// 先创建身份
	identity := createIdentityForTest(t, r, "爸爸")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/identities/"+identity.ID+"/playlists", nil)
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

func TestHandler_CreatePlaylist(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	identity := createIdentityForTest(t, r, "爸爸")

	body := map[string]interface{}{
		"name":       "通勤",
		"sort_order": 0,
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/identities/"+identity.ID+"/playlists", bytes.NewBuffer(jsonBody))
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

func TestHandler_CreatePlaylist_InvalidIdentity(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	body := map[string]interface{}{
		"name": "通勤",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/identities/nonexistent/playlists", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("状态码错误: got %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandler_CreatePlaylist_Validation(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	identity := createIdentityForTest(t, r, "爸爸")

	body := map[string]interface{}{
		"name": "",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/identities/"+identity.ID+"/playlists", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("状态码错误: got %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp model.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Code != ErrCodePlaylistNameEmpty {
		t.Errorf("错误码错误: got %d, want %d", resp.Code, ErrCodePlaylistNameEmpty)
	}
}

func TestHandler_GetPlaylist_NotFound(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/playlists/nonexistent", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("状态码错误: got %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandler_DeletePlaylist(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	identity := createIdentityForTest(t, r, "爸爸")
	playlist := createPlaylistForTest(t, r, identity.ID, "通勤")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/playlists/"+playlist.ID, nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码错误: got %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// createIdentityForTest 测试辅助：创建身份
func createIdentityForTest(t *testing.T, r http.Handler, name string) *model.Identity {
	t.Helper()
	body := map[string]interface{}{
		"name":         name,
		"avatar_color": "#6366f1",
	}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/identities", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	var resp model.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析身份响应失败: %v", err)
	}
	data, _ := json.Marshal(resp.Data)
	var identity model.Identity
	_ = json.Unmarshal(data, &identity)
	return &identity
}

// createPlaylistForTest 测试辅助：创建歌单
func createPlaylistForTest(t *testing.T, r http.Handler, identityID, name string) *model.Playlist {
	t.Helper()
	body := map[string]interface{}{
		"name":       name,
		"sort_order": 0,
	}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/identities/"+identityID+"/playlists", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	var resp model.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析歌单响应失败: %v", err)
	}
	data, _ := json.Marshal(resp.Data)
	var playlist model.Playlist
	_ = json.Unmarshal(data, &playlist)
	return &playlist
}
