package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/ety001/multitune/internal/config"
	"github.com/ety001/multitune/internal/db"
	"github.com/ety001/multitune/internal/model"
	"github.com/gin-gonic/gin"
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

func TestHandler_DeletePlaylist_NotFound(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/playlists/nonexistent", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("状态码错误: got %d, want %d", w.Code, http.StatusNotFound)
	}

	var resp model.APIResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != ErrCodePlaylistNotFound {
		t.Errorf("错误码错误: got %d, want %d", resp.Code, ErrCodePlaylistNotFound)
	}
}

func TestHandler_UpdatePlaylist(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	identity := createIdentityForTest(t, r, "爸爸")
	playlist := createPlaylistForTest(t, r, identity.ID, "通勤")

	updateBody := map[string]interface{}{
		"name":       "通勤-改",
		"sort_order": 5,
	}
	jsonBody, _ := json.Marshal(updateBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/playlists/"+playlist.ID, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码错误: got %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp model.APIResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data, _ := json.Marshal(resp.Data)
	var updated model.Playlist
	_ = json.Unmarshal(data, &updated)

	if updated.Name != "通勤-改" {
		t.Errorf("名称未更新: got %s, want 通勤-改", updated.Name)
	}
	if updated.SortOrder != 5 {
		t.Errorf("排序未更新: got %d, want 5", updated.SortOrder)
	}
}

func TestHandler_GetPlaylist(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	identity := createIdentityForTest(t, r, "爸爸")
	playlist := createPlaylistForTest(t, r, identity.ID, "通勤")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/playlists/"+playlist.ID, nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码错误: got %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp model.APIResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 0 {
		t.Errorf("响应码错误: got %d, want 0", resp.Code)
	}
}

func TestHandler_CreatePlaylist_MaxLimit(t *testing.T) {
	cfg := &config.Config{
		DataPath:                t.TempDir(),
		DatabaseName:            "test.db",
		MaxIdentities:           20,
		MaxPlaylistsPerIdentity: 2,
		MaxSongsPerPlaylist:     1000,
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

	create := func(name string) int {
		body := map[string]interface{}{
			"name": name,
		}
		jsonBody, _ := json.Marshal(body)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/identities/"+identity.ID+"/playlists", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		return w.Code
	}

	// 前 2 个成功
	if code := create("A"); code != http.StatusOK {
		t.Errorf("第1个应成功: got %d", code)
	}
	if code := create("B"); code != http.StatusOK {
		t.Errorf("第2个应成功: got %d", code)
	}
	// 第 3 个被拒绝
	if code := create("C"); code != http.StatusBadRequest {
		t.Errorf("第3个应被拒绝: got %d, want %d", code, http.StatusBadRequest)
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
