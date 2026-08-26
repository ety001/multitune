package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ety001/multitune/internal/model"
)

func TestScanJobRepo_List(t *testing.T) {
	h := newTestHandler(t)

	identity, err := h.identityRepo.Create("测试身份", "#000000", 0)
	if err != nil {
		t.Fatal(err)
	}
	pl1, err := h.playlistRepo.Create(identity.ID, "歌单一", 0)
	if err != nil {
		t.Fatal(err)
	}
	pl2, err := h.playlistRepo.Create(identity.ID, "歌单二", 1)
	if err != nil {
		t.Fatal(err)
	}

	job1, err := h.scanJobRepo.Create(pl1.ID, []string{"/music/a", "/music/b"})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	job2, err := h.scanJobRepo.Create(pl2.ID, nil)
	if err != nil {
		t.Fatal(err)
	}

	jobs, err := h.scanJobRepo.List(20)
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("应有 2 个任务, got %d", len(jobs))
	}
	// 按创建时间倒序：最新的在前
	if jobs[0].ID != job2.ID || jobs[1].ID != job1.ID {
		t.Errorf("应按创建时间倒序: got %s, %s", jobs[0].ID, jobs[1].ID)
	}
	if jobs[1].PlaylistName != "歌单一" {
		t.Errorf("歌单名关联错误: got %q", jobs[1].PlaylistName)
	}
	// paths 持久化与还原；nil 归一为空列表
	if len(jobs[1].Paths) != 2 || jobs[1].Paths[0] != "/music/a" {
		t.Errorf("paths 还原错误: got %v", jobs[1].Paths)
	}
	if jobs[0].Paths == nil || len(jobs[0].Paths) != 0 {
		t.Errorf("nil paths 应归一为空列表: got %v", jobs[0].Paths)
	}

	// GetByID 也应带出 paths
	got, err := h.scanJobRepo.GetByID(job1.ID)
	if err != nil || got == nil {
		t.Fatalf("GetByID 失败: %v", err)
	}
	if len(got.Paths) != 2 {
		t.Errorf("GetByID paths 还原错误: got %v", got.Paths)
	}
}

func TestHandler_ListScanJobs(t *testing.T) {
	h := newTestHandler(t)
	r := h.SetupRouter()

	// 空库：应返回 200 + items:[]（不是 null）
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/scan/jobs", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("状态码错误: got %d, body: %s", w.Code, w.Body.String())
	}
	if !json.Valid(w.Body.Bytes()) {
		t.Fatal("响应不是合法 JSON")
	}
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Items []model.ScanJobWithPlaylist `json:"items"`
			Total int                         `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Code != 0 || resp.Data.Total != 0 || len(resp.Data.Items) != 0 {
		t.Errorf("空列表应返回 items:[]: code=%d total=%d items=%v", resp.Code, resp.Data.Total, resp.Data.Items)
	}

	// 有数据：正常返回
	identity, _ := h.identityRepo.Create("身份", "#000000", 0)
	playlist, _ := h.playlistRepo.Create(identity.ID, "我的歌单", 0)
	if _, err := h.scanJobRepo.Create(playlist.ID, []string{"/music"}); err != nil {
		t.Fatal(err)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/scan/jobs?limit=5", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("状态码错误: got %d", w.Code)
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Data.Total != 1 {
		t.Fatalf("应有 1 个任务, got %d", resp.Data.Total)
	}
	if resp.Data.Items[0].PlaylistName != "我的歌单" {
		t.Errorf("歌单名错误: got %q", resp.Data.Items[0].PlaylistName)
	}
	if len(resp.Data.Items[0].Paths) != 1 || resp.Data.Items[0].Paths[0] != "/music" {
		t.Errorf("paths 错误: got %v", resp.Data.Items[0].Paths)
	}
}
