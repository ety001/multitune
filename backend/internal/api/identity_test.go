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

func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	cfg := &config.Config{
		DataPath:                t.TempDir(),
		DatabaseName:            "test.db",
		MaxIdentities:           20,
		MaxPlaylistsPerIdentity: 50,
		MaxSongsPerPlaylist:     1000,
		GINMode:                 gin.TestMode,
		StaticPath:              "/nonexistent",
		ScanFormats:             []string{"mp3", "flac", "m4a", "aac", "ogg", "wav"},
	}
	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	t.Cleanup(func() {
		_ = database.Close()
	})

	// 关闭默认日志输出
	gin.DefaultWriter = os.NewFile(0, os.DevNull)
	return NewHandler(cfg, database)
}

func TestHandler_ListIdentities(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/identities", nil)
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

func TestHandler_CreateIdentity(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	body := map[string]interface{}{
		"name":         "爸爸",
		"avatar_color": "#6366f1",
		"sort_order":   0,
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/identities", bytes.NewBuffer(jsonBody))
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

func TestHandler_CreateIdentity_Validation(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	body := map[string]interface{}{
		"name": "",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/identities", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("状态码错误: got %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp model.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Code != ErrCodeIdentityNameEmpty {
		t.Errorf("错误码错误: got %d, want %d", resp.Code, ErrCodeIdentityNameEmpty)
	}
}

func TestHandler_SetDefaultIdentity(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	// 创建两个身份
	create := func(name string) string {
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
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		data, _ := json.Marshal(resp.Data)
		var identity model.Identity
		_ = json.Unmarshal(data, &identity)
		return identity.ID
	}

	id := create("爸爸")
	create("妈妈")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/identities/"+id+"/default", nil)
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

func TestHandler_GetIdentity_NotFound(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/identities/nonexistent", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("状态码错误: got %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandler_UpdateIdentity(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	// 先创建身份
	createBody := map[string]interface{}{
		"name":         "爸爸",
		"avatar_color": "#6366f1",
	}
	jsonBody, _ := json.Marshal(createBody)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/identities", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	var resp model.APIResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data, _ := json.Marshal(resp.Data)
	var identity model.Identity
	_ = json.Unmarshal(data, &identity)

	// 更新名称和排序
	updateBody := map[string]interface{}{
		"name":       "爸爸-改",
		"sort_order": 5,
	}
	jsonBody, _ = json.Marshal(updateBody)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/api/identities/"+identity.ID, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码错误: got %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data, _ = json.Marshal(resp.Data)
	var updated model.Identity
	_ = json.Unmarshal(data, &updated)

	if updated.Name != "爸爸-改" {
		t.Errorf("名称未更新: got %s, want 爸爸-改", updated.Name)
	}
	if updated.SortOrder != 5 {
		t.Errorf("排序未更新: got %d, want 5", updated.SortOrder)
	}
}

func TestHandler_DeleteIdentity(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	// 先创建
	createBody := map[string]interface{}{
		"name":         "爸爸",
		"avatar_color": "#6366f1",
	}
	jsonBody, _ := json.Marshal(createBody)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/identities", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	var resp model.APIResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data, _ := json.Marshal(resp.Data)
	var identity model.Identity
	_ = json.Unmarshal(data, &identity)

	// 删除
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/api/identities/"+identity.ID, nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("删除状态码错误: got %d, want %d", w.Code, http.StatusOK)
	}

	// 确认已删除
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/identities/"+identity.ID, nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("删除后应返回 404: got %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandler_DeleteIdentity_NotFound(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/identities/nonexistent", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("状态码错误: got %d, want %d", w.Code, http.StatusNotFound)
	}

	var resp model.APIResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != ErrCodeIdentityNotFound {
		t.Errorf("错误码错误: got %d, want %d", resp.Code, ErrCodeIdentityNotFound)
	}
}

func TestHandler_CreateIdentity_MaxLimit(t *testing.T) {
	cfg := &config.Config{
		DataPath:                t.TempDir(),
		DatabaseName:            "test.db",
		MaxIdentities:           2,
		MaxPlaylistsPerIdentity: 50,
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

	create := func(name string) int {
		body := map[string]interface{}{
			"name":         name,
			"avatar_color": "#6366f1",
		}
		jsonBody, _ := json.Marshal(body)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/identities", bytes.NewBuffer(jsonBody))
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

func TestHandler_CreateIdentity_DefaultAvatarColor(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	// 不传 avatar_color
	body := map[string]interface{}{
		"name": "无颜色",
	}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/identities", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码错误: got %d, body: %s", w.Code, w.Body.String())
	}

	var resp model.APIResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data, _ := json.Marshal(resp.Data)
	var identity model.Identity
	_ = json.Unmarshal(data, &identity)

	if identity.AvatarColor != "#6366f1" {
		t.Errorf("默认颜色错误: got %s, want #6366f1", identity.AvatarColor)
	}
}
