package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ety001/multitune/internal/config"
	"github.com/ety001/multitune/internal/db"
	"github.com/ety001/multitune/internal/model"
	"github.com/gin-gonic/gin"
)

// newLazycatTestHandler 构造懒猫部署模式的测试 handler，
// 文档根与媒体根指向临时目录，模拟 /lzcapp/document 与 /lzcapp/media：
//
//	<tmp>/document/alice/music/a.mp3
//	<tmp>/document/bob/music/b.mp3
//	<tmp>/media/media/music/u.mp3            （USB 挂载）
//	<tmp>/media/RemoteFS/alice/music/r.mp3    （远程挂载-alice）
//	<tmp>/media/RemoteFS/bob/music/rb.mp3     （远程挂载-bob）
func newLazycatTestHandler(t *testing.T) (*Handler, string, string) {
	t.Helper()

	base := t.TempDir()
	docRoot := filepath.Join(base, "document")
	mediaRoot := filepath.Join(base, "media")
	for _, dir := range []string{
		filepath.Join(docRoot, "alice", "music"),
		filepath.Join(docRoot, "bob", "music"),
		filepath.Join(mediaRoot, "media", "music"),
		filepath.Join(mediaRoot, "RemoteFS", "alice", "music"),
		filepath.Join(mediaRoot, "RemoteFS", "bob", "music"),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []struct {
		path    string
		content string
	}{
		{filepath.Join(docRoot, "alice", "music", "a.mp3"), "a"},
		{filepath.Join(docRoot, "bob", "music", "b.mp3"), "b"},
		{filepath.Join(mediaRoot, "media", "music", "u.mp3"), "u"},
		{filepath.Join(mediaRoot, "RemoteFS", "alice", "music", "r.mp3"), "r"},
		{filepath.Join(mediaRoot, "RemoteFS", "bob", "music", "rb.mp3"), "rb"},
	} {
		if err := os.WriteFile(f.path, []byte(f.content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	cfg := &config.Config{
		DataPath:                t.TempDir(),
		CachePath:               filepath.Join(t.TempDir(), "cache"),
		DatabaseName:            "test.db",
		MaxIdentities:           20,
		MaxPlaylistsPerIdentity: 50,
		MaxSongsPerPlaylist:     1000,
		ScanFormats:             []string{"mp3", "flac", "m4a", "aac", "ogg", "wav"},
		GINMode:                 gin.TestMode,
		StaticPath:              "/nonexistent",
		LazyCatDeploy:           true,
		LazyCatDocumentRoot:     docRoot,
		LazyCatMediaRoot:        mediaRoot,
	}
	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	t.Cleanup(func() {
		_ = database.Close()
	})

	gin.DefaultWriter = os.NewFile(0, os.DevNull)
	return NewHandler(cfg, database), docRoot, mediaRoot
}

func TestLazycat_ListStorageSources(t *testing.T) {
	h, docRoot, mediaRoot := newLazycatTestHandler(t)
	r := h.SetupRouter()

	// 无用户身份：拒绝
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/fs/sources", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("无用户名应返回 401，got %d, body: %s", w.Code, w.Body.String())
	}

	// 伪造带路径成分的用户名：拒绝
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/fs/sources", nil)
	req.Header.Set("X-HC-User-ID", "../etc")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("非法用户名应返回 401，got %d", w.Code)
	}

	// 正常用户：返回媒体 + 自己的文档目录
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/fs/sources", nil)
	req.Header.Set("X-HC-User-ID", "alice")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("状态码错误: got %d, body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Items []struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				Path      string `json:"path"`
				Available bool   `json:"available"`
			} `json:"items"`
			Total int `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Code != 0 || resp.Data.Total != 3 {
		t.Fatalf("应返回 3 个存储源，got code=%d total=%d", resp.Code, resp.Data.Total)
	}
	byID := map[string]string{}
	for _, s := range resp.Data.Items {
		byID[s.ID] = s.Path
	}
	if byID["remotefs"] != filepath.Join(mediaRoot, "RemoteFS", "alice") {
		t.Errorf("远程挂载源路径错误: got %s, want %s", byID["remotefs"], filepath.Join(mediaRoot, "RemoteFS", "alice"))
	}
	if byID["usb"] != filepath.Join(mediaRoot, "media") {
		t.Errorf("USB 挂载源路径错误: got %s, want %s", byID["usb"], filepath.Join(mediaRoot, "media"))
	}
	if byID["document"] != filepath.Join(docRoot, "alice") {
		t.Errorf("文档源路径错误: got %s, want %s", byID["document"], filepath.Join(docRoot, "alice"))
	}
	for _, s := range resp.Data.Items {
		if !s.Available {
			t.Errorf("源 %s 应可用", s.ID)
		}
	}
}

func TestLazycat_ListStorageSources_UsernameOverride(t *testing.T) {
	h, docRoot, _ := newLazycatTestHandler(t)
	h.cfg.LazyCatUsername = "carol" // 开发机直连调试时的模拟用户
	r := h.SetupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/fs/sources", nil)
	req.Header.Set("X-HC-User-ID", "alice")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("状态码错误: got %d", w.Code)
	}

	var resp struct {
		Data struct {
			Items []struct {
				ID   string `json:"id"`
				Path string `json:"path"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	for _, s := range resp.Data.Items {
		if s.ID == "document" && s.Path != filepath.Join(docRoot, "carol") {
			t.Errorf("配置覆盖未生效: got %s, want %s", s.Path, filepath.Join(docRoot, "carol"))
		}
	}
}

func TestLazycat_ListDirectory(t *testing.T) {
	h, docRoot, mediaRoot := newLazycatTestHandler(t)
	r := h.SetupRouter()

	cases := []struct {
		name string
		user string
		path string
		want int
	}{
		{"访问自己的文档目录", "alice", filepath.Join(docRoot, "alice", "music"), http.StatusOK},
		{"访问自己的文档根", "alice", filepath.Join(docRoot, "alice"), http.StatusOK},
		{"访问别人的文档目录", "alice", filepath.Join(docRoot, "bob", "music"), http.StatusBadRequest},
		{"访问别人的文档根", "alice", filepath.Join(docRoot, "bob"), http.StatusBadRequest},
		{"访问文档总根", "alice", docRoot, http.StatusBadRequest},
		{"前缀相似目录不可绕过", "alice", filepath.Join(docRoot, "alice2"), http.StatusBadRequest},
		{"访问自己的远程挂载目录", "alice", filepath.Join(mediaRoot, "RemoteFS", "alice", "music"), http.StatusOK},
		{"访问别人的远程挂载目录", "alice", filepath.Join(mediaRoot, "RemoteFS", "bob", "music"), http.StatusBadRequest},
		{"访问远程挂载总根", "alice", filepath.Join(mediaRoot, "RemoteFS"), http.StatusBadRequest},
		{"访问USB挂载目录", "alice", filepath.Join(mediaRoot, "media", "music"), http.StatusOK},
		{"媒体总根已不可访问", "alice", mediaRoot, http.StatusBadRequest},
		{"访问 /lzcapp 上级", "alice", filepath.Dir(docRoot), http.StatusBadRequest},
		{"访问根路径", "alice", "/", http.StatusOK},
		{"无用户身份", "", filepath.Join(mediaRoot, "media", "music"), http.StatusUnauthorized},
		{"路径穿越逃逸", "alice", docRoot + "/alice/../../bob/music", http.StatusBadRequest},
		{"远程挂载前缀相似不可绕过", "alice", filepath.Join(mediaRoot, "RemoteFS", "alice2"), http.StatusBadRequest},
		{"尾部斜杠仍允许", "alice", filepath.Join(docRoot, "alice") + "/", http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/fs/list?path="+tc.path, nil)
			if tc.user != "" {
				req.Header.Set("X-HC-User-ID", tc.user)
			}
			r.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("状态码错误: got %d, want %d, body: %s", w.Code, tc.want, w.Body.String())
			}
		})
	}
}

func TestLazycat_ListDirectory_DefaultsToFirstRootAndClampsParent(t *testing.T) {
	h, docRoot, mediaRoot := newLazycatTestHandler(t)
	r := h.SetupRouter()

	// path="/" 时落到第一个允许根（远程挂载的用户目录）
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/fs/list?path=/", nil)
	req.Header.Set("X-HC-User-ID", "alice")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("状态码错误: got %d, body: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Path string `json:"path"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Data.Path != filepath.Join(mediaRoot, "RemoteFS", "alice") {
		t.Errorf("根路径应落到媒体目录: got %s, want %s", resp.Data.Path, mediaRoot)
	}

	// 在自己文档根下，上级已越出允许范围，parent 应夹紧为当前目录（前端禁用"上级"）
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/fs/list?path="+filepath.Join(docRoot, "alice"), nil)
	req.Header.Set("X-HC-User-ID", "alice")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("状态码错误: got %d, body: %s", w.Code, w.Body.String())
	}
	var listResp struct {
		Data struct {
			Path   string `json:"path"`
			Parent string `json:"parent"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if listResp.Data.Parent != listResp.Data.Path {
		t.Errorf("上级越出允许范围时应夹紧: parent=%s path=%s", listResp.Data.Parent, listResp.Data.Path)
	}
}

func TestLazycat_ScanSongs(t *testing.T) {
	h, docRoot, mediaRoot := newLazycatTestHandler(t)
	r := h.SetupRouter()

	doScan := func(user, path string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]string{"path": path})
		req, _ := http.NewRequest("POST", "/api/scan", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		if user != "" {
			req.Header.Set("X-HC-User-ID", user)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	// 扫描别人的文档目录：拒绝
	if w := doScan("alice", filepath.Join(docRoot, "bob", "music")); w.Code != http.StatusBadRequest {
		t.Errorf("扫描他人目录应返回 400，got %d, body: %s", w.Code, w.Body.String())
	}
	// 扫描别人的远程挂载目录：拒绝
	if w := doScan("alice", filepath.Join(mediaRoot, "RemoteFS", "bob", "music")); w.Code != http.StatusBadRequest {
		t.Errorf("扫描他人远程挂载应返回 400，got %d, body: %s", w.Code, w.Body.String())
	}
	// 媒体总根已不再是允许根：拒绝
	if w := doScan("alice", mediaRoot); w.Code != http.StatusBadRequest {
		t.Errorf("扫描媒体总根应返回 400，got %d", w.Code)
	}
	// 无用户身份：拒绝
	if w := doScan("", filepath.Join(mediaRoot, "media", "music")); w.Code != http.StatusUnauthorized {
		t.Errorf("无用户名应返回 401，got %d", w.Code)
	}
	// 扫描 USB 挂载目录：允许并成功入库
	if w := doScan("alice", filepath.Join(mediaRoot, "media", "music")); w.Code != http.StatusOK {
		t.Errorf("扫描 USB 挂载应返回 200，got %d, body: %s", w.Code, w.Body.String())
	}
	// 扫描自己的远程挂载目录：允许
	if w := doScan("alice", filepath.Join(mediaRoot, "RemoteFS", "alice", "music")); w.Code != http.StatusOK {
		t.Errorf("扫描自己远程挂载应返回 200，got %d, body: %s", w.Code, w.Body.String())
	}
	// 扫描自己的文档目录：允许
	if w := doScan("alice", filepath.Join(docRoot, "alice", "music")); w.Code != http.StatusOK {
		t.Errorf("扫描自己目录应返回 200，got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestLazycat_CreateScanJob(t *testing.T) {
	h, docRoot, mediaRoot := newLazycatTestHandler(t)
	r := h.SetupRouter()

	identity, err := h.identityRepo.Create("测试身份", "#000000", 0)
	if err != nil {
		t.Fatal(err)
	}
	playlist, err := h.playlistRepo.Create(identity.ID, "测试歌单", 0)
	if err != nil {
		t.Fatal(err)
	}

	doCreate := func(user string, paths []string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]interface{}{"paths": paths, "playlist_id": playlist.ID})
		req, _ := http.NewRequest("POST", "/api/scan/jobs", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		if user != "" {
			req.Header.Set("X-HC-User-ID", user)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	// 全部路径合法：创建成功
	w := doCreate("alice", []string{
		filepath.Join(docRoot, "alice", "music"),
		filepath.Join(mediaRoot, "media", "music"),
		filepath.Join(mediaRoot, "RemoteFS", "alice", "music"),
	})
	if w.Code != http.StatusOK {
		t.Fatalf("合法路径应返回 200，got %d, body: %s", w.Code, w.Body.String())
	}
	var jobResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &jobResp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	// 等待异步扫描结束，避免测试清理与后台任务竞态
	for i := 0; i < 100; i++ {
		wj := httptest.NewRecorder()
		reqj, _ := http.NewRequest("GET", "/api/scan/jobs/"+jobResp.Data.ID, nil)
		r.ServeHTTP(wj, reqj)
		var jr struct {
			Data struct {
				Status string `json:"status"`
			} `json:"data"`
		}
		if err := json.Unmarshal(wj.Body.Bytes(), &jr); err != nil {
			t.Fatalf("解析任务响应失败: %v", err)
		}
		if jr.Data.Status == "done" || jr.Data.Status == "error" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	// 混入他人目录：整单拒绝
	if w := doCreate("alice", []string{filepath.Join(docRoot, "alice", "music"), filepath.Join(docRoot, "bob", "music")}); w.Code != http.StatusBadRequest {
		t.Errorf("混入他人目录应返回 400，got %d", w.Code)
	}

	// 无用户身份：拒绝
	if w := doCreate("", []string{filepath.Join(mediaRoot, "music")}); w.Code != http.StatusUnauthorized {
		t.Errorf("无用户名应返回 401，got %d", w.Code)
	}
}

// 确认响应体的业务码符合约定
func TestLazycat_ErrorCodes(t *testing.T) {
	h, docRoot, _ := newLazycatTestHandler(t)
	r := h.SetupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/fs/list?path="+filepath.Join(docRoot, "bob"), nil)
	req.Header.Set("X-HC-User-ID", "alice")
	r.ServeHTTP(w, req)

	var resp model.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Code != ErrCodePathNotAccessible {
		t.Errorf("越权路径业务码错误: got %d, want %d", resp.Code, ErrCodePathNotAccessible)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/fs/sources", nil)
	r.ServeHTTP(w, req)
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Code != ErrCodeUserNotIdentified {
		t.Errorf("无身份业务码错误: got %d, want %d", resp.Code, ErrCodeUserNotIdentified)
	}
}
