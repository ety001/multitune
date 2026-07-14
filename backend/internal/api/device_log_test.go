package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ety001/multitune/internal/model"
)

func TestHandler_CreateDeviceLog(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	body := map[string]interface{}{
		"userAgent":      "Mozilla/5.0 Chrome/74.0",
		"chromeVersion":  74,
		"webViewVersion": 74,
		"isWebView":      true,
		"screenWidth":    1920,
		"screenHeight":   1080,
		"windowWidth":    1920,
		"windowHeight":   1080,
		"language":       "zh-CN",
		"platform":       "Linux armv8l",
		"cookieEnabled":  true,
		"onLine":         true,
		"timestamp":      "2026-07-14T10:00:00Z",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/device-info", bytes.NewBuffer(jsonBody))
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
	var log model.DeviceLog
	_ = json.Unmarshal(data, &log)
	if log.ID == 0 {
		t.Errorf("返回日志 ID 不应为 0")
	}
	if log.ChromeVersion != 74 {
		t.Errorf("chrome_version = %d, want 74", log.ChromeVersion)
	}
	if !log.IsWebview {
		t.Errorf("is_webview = false, want true")
	}
}

func TestHandler_CreateDeviceLog_InvalidJSON(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/device-info", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("状态码错误: got %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandler_ListDeviceLogs(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	// 先创建两条日志
	for i := 0; i < 2; i++ {
		body := map[string]interface{}{
			"userAgent":     "agent",
			"chromeVersion": 70 + i,
			"platform":      "Linux",
		}
		jsonBody, _ := json.Marshal(body)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/device-info", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("预创建日志失败: %d", w.Code)
		}
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/device-logs?limit=10&offset=0", nil)
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
	var list model.ListResponse
	_ = json.Unmarshal(data, &list)
	if list.Total != 2 {
		t.Errorf("total = %d, want 2", list.Total)
	}

	items, ok := list.Items.([]interface{})
	if !ok {
		t.Fatalf("items 类型错误")
	}
	if len(items) != 2 {
		t.Errorf("items 长度 = %d, want 2", len(items))
	}
}

func TestHandler_ListDeviceLogs_Empty(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/device-logs", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码错误: got %d, want %d", w.Code, http.StatusOK)
	}

	var resp model.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	data, _ := json.Marshal(resp.Data)
	var list model.ListResponse
	_ = json.Unmarshal(data, &list)
	if list.Total != 0 {
		t.Errorf("total = %d, want 0", list.Total)
	}
	items, ok := list.Items.([]interface{})
	if !ok {
		t.Fatalf("items 类型错误")
	}
	if len(items) != 0 {
		t.Errorf("空列表 items 长度 = %d, want 0", len(items))
	}
}
