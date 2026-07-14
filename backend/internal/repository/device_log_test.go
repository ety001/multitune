package repository

import (
	"testing"

	"github.com/ety001/multitune/internal/model"
)

func TestDeviceLogRepo_CreateAndGet(t *testing.T) {
	database := newTestDB(t)
	r := NewDeviceLogRepo(database)

	log := &model.DeviceLog{
		UserAgent:      "Mozilla/5.0 Chrome/74.0",
		ChromeVersion:  74,
		WebviewVersion: 74,
		IsWebview:      true,
		ScreenWidth:    1920,
		ScreenHeight:   1080,
		WindowWidth:    1920,
		WindowHeight:   1080,
		Language:       "zh-CN",
		Platform:       "Linux armv8l",
		CookieEnabled:  true,
		Online:         true,
		Timestamp:      "2026-07-14T10:00:00Z",
	}

	created, err := r.Create(log)
	if err != nil {
		t.Fatalf("创建设备日志失败: %v", err)
	}
	if created.ID == 0 {
		t.Errorf("创建后 ID 不应为 0")
	}
	if created.CreatedAt == 0 {
		t.Errorf("创建后 created_at 不应为 0")
	}
	if created.UserAgent != log.UserAgent {
		t.Errorf("user_agent = %q, want %q", created.UserAgent, log.UserAgent)
	}

	got, err := r.GetByID(created.ID)
	if err != nil {
		t.Fatalf("查询设备日志失败: %v", err)
	}
	if got == nil {
		t.Fatalf("应能查到刚创建的日志")
	}
	if got.ChromeVersion != 74 {
		t.Errorf("chrome_version = %d, want 74", got.ChromeVersion)
	}
	if !got.IsWebview {
		t.Errorf("is_webview = false, want true")
	}
}

func TestDeviceLogRepo_GetByID_NotFound(t *testing.T) {
	database := newTestDB(t)
	r := NewDeviceLogRepo(database)

	got, err := r.GetByID(999999)
	if err != nil {
		t.Fatalf("查询不存在日志不应报错: %v", err)
	}
	if got != nil {
		t.Errorf("不存在的日志应返回 nil, got %+v", got)
	}
}

func TestDeviceLogRepo_List(t *testing.T) {
	database := newTestDB(t)
	r := NewDeviceLogRepo(database)

	for i := 0; i < 3; i++ {
		_, err := r.Create(&model.DeviceLog{
			UserAgent:     "agent",
			ChromeVersion: 70 + i,
			Platform:      "Linux",
		})
		if err != nil {
			t.Fatalf("创建设备日志失败: %v", err)
		}
	}

	logs, total, err := r.List(2, 0)
	if err != nil {
		t.Fatalf("查询列表失败: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if len(logs) != 2 {
		t.Errorf("len(logs) = %d, want 2", len(logs))
	}

	logs, total, err = r.List(10, 10)
	if err != nil {
		t.Fatalf("查询 offset 超出失败: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if len(logs) != 0 {
		t.Errorf("offset 超出应返回空列表, got %d", len(logs))
	}
}

func TestDeviceLogRepo_List_Empty(t *testing.T) {
	database := newTestDB(t)
	r := NewDeviceLogRepo(database)

	logs, total, err := r.List(20, 0)
	if err != nil {
		t.Fatalf("空列表查询失败: %v", err)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if len(logs) != 0 {
		t.Errorf("len(logs) = %d, want 0", len(logs))
	}
}
