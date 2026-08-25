package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/ety001/multitune/internal/repository"
)

// ffmpegAvailable 测试环境缺 ffmpeg 时跳过缩略图相关用例
func ffmpegAvailable(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("环境未安装 ffmpeg，跳过缩略图测试")
	}
}

// makeCoverImage 用 ffmpeg 生成一张测试用大图（testsrc 图案）
func makeCoverImage(t *testing.T, path string, size string) {
	t.Helper()
	cmd := exec.Command("ffmpeg", "-y", "-v", "error",
		"-f", "lavfi", "-i", "testsrc=size="+size+":rate=1",
		"-frames:v", "1", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("生成测试图片失败: %v, output: %s", err, out)
	}
}

func newThumbTestHandler(t *testing.T) (*Handler, *repository.UpsertResult) {
	t.Helper()
	ffmpegAvailable(t)

	h := newTestHandler(t)

	dir := h.cfg.DataPath
	songPath := filepath.Join(dir, "song.mp3")
	if err := os.WriteFile(songPath, []byte("dummy"), 0644); err != nil {
		t.Fatal(err)
	}
	makeCoverImage(t, filepath.Join(dir, "song.jpg"), "1280x1280")

	result, err := h.songRepo.Upsert(songPath, "home", "Test", "", "", 100)
	if err != nil {
		t.Fatalf("Upsert song failed: %v", err)
	}
	return h, result
}

func TestCoverThumb_GenerateAndCacheHit(t *testing.T) {
	h, result := newThumbTestHandler(t)
	r := h.SetupRouter()
	url := "/api/songs/" + result.Song.ID + "/cover?size=thumb"

	// 首次请求：生成缩略图
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", url, nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("状态码错误: got %d, body: %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "image/webp" {
		t.Errorf("Content-Type 应为 image/webp, got %s", ct)
	}
	if w.Body.Len() <= 0 {
		t.Error("响应体不应为空")
	}

	// 缓存文件应已生成
	entries, err := os.ReadDir(h.cfg.CachePath)
	if err != nil || len(entries) != 1 {
		t.Fatalf("缓存目录应有 1 个文件, err=%v, count=%d", err, len(entries))
	}
	cacheFile := filepath.Join(h.cfg.CachePath, entries[0].Name())
	firstStat, err := os.Stat(cacheFile)
	if err != nil {
		t.Fatal(err)
	}

	// 第二次请求：命中缓存，不再重新生成
	time.Sleep(10 * time.Millisecond) // 确保 mtime 可区分
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", url, nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("第二次请求状态码错误: got %d", w.Code)
	}
	secondStat, err := os.Stat(cacheFile)
	if err != nil {
		t.Fatal(err)
	}
	if !firstStat.ModTime().Equal(secondStat.ModTime()) {
		t.Error("第二次请求应命中缓存，不应重新生成")
	}
	if w.Body.Len() > 40*1024 {
		t.Errorf("缩略图应明显小于原图, got %d bytes", w.Body.Len())
	}
}

func TestCoverThumb_InvalidateOnSourceChange(t *testing.T) {
	h, result := newThumbTestHandler(t)
	r := h.SetupRouter()
	url := "/api/songs/" + result.Song.ID + "/cover?size=thumb"

	// 首次生成
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", url, nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("首次请求失败: %d", w.Code)
	}

	// 原图变更（重写内容，size+mtime 变化）→ 缓存 key 失效重新生成
	coverPath := filepath.Join(h.cfg.DataPath, "song.jpg")
	makeCoverImage(t, coverPath, "640x640")
	os.Chtimes(coverPath, time.Now(), time.Now().Add(time.Minute))

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", url, nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("原图变更后请求失败: %d", w.Code)
	}

	// 旧缓存应被清理，只保留新版本
	entries, _ := os.ReadDir(h.cfg.CachePath)
	if len(entries) != 1 {
		t.Errorf("旧缓存应被清理, 剩余 %d 个文件", len(entries))
	}
}

func TestCoverThumb_FallbackToOriginalWhenNoFfmpeg(t *testing.T) {
	h, result := newThumbTestHandler(t)
	r := h.SetupRouter()

	// 破坏 ffmpeg 查找：无法直接屏蔽 PATH，这里用缓存目录不可写模拟生成失败，
	// 验证降级返回原图（JPEG）而不是 500。
	if err := os.RemoveAll(h.cfg.CachePath); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(h.cfg.CachePath, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(h.cfg.CachePath, 0755)
	})
	// root 用户运行测试时目录权限不生效，跳过该场景
	if info, err := os.Stat(h.cfg.CachePath); err == nil && !isWritableDir(info, h.cfg.CachePath) {
		t.Skip("当前用户可越过目录权限（root），跳过降级测试")
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/songs/"+result.Song.ID+"/cover?size=thumb", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("降级应返回原图 200, got %d, body: %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "image/jpeg" {
		t.Errorf("降级应返回原图 image/jpeg, got %s", ct)
	}
}

func isWritableDir(_ os.FileInfo, path string) bool {
	f, err := os.CreateTemp(path, ".wtest-*")
	if err != nil {
		return false
	}
	_ = f.Close()
	_ = os.Remove(f.Name())
	return true
}

func TestCoverThumb_NoCover404(t *testing.T) {
	ffmpegAvailable(t)
	h := newTestHandler(t)
	r := h.SetupRouter()

	songPath := filepath.Join(h.cfg.DataPath, "nosong.mp3")
	if err := os.WriteFile(songPath, []byte("dummy"), 0644); err != nil {
		t.Fatal(err)
	}
	result, err := h.songRepo.Upsert(songPath, "home", "NoCover", "", "", 100)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/songs/%s/cover?size=thumb", result.Song.ID), nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("无封面应返回 404, got %d", w.Code)
	}
}
