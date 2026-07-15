-- 设备信息日志表
CREATE TABLE IF NOT EXISTS device_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_agent TEXT,
    chrome_version INTEGER DEFAULT 0,
    webview_version INTEGER DEFAULT 0,
    is_webview INTEGER NOT NULL DEFAULT 0 CHECK(is_webview IN (0, 1)),
    screen_width INTEGER DEFAULT 0,
    screen_height INTEGER DEFAULT 0,
    window_width INTEGER DEFAULT 0,
    window_height INTEGER DEFAULT 0,
    language TEXT,
    platform TEXT,
    cookie_enabled INTEGER NOT NULL DEFAULT 0 CHECK(cookie_enabled IN (0, 1)),
    online INTEGER NOT NULL DEFAULT 0 CHECK(online IN (0, 1)),
    timestamp TEXT,
    created_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_device_logs_created_at ON device_logs(created_at DESC);
