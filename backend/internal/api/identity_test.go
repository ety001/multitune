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
