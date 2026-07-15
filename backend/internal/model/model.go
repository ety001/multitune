package model

// Identity 身份
type Identity struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	AvatarColor string `json:"avatar_color"`
	SortOrder   int    `json:"sort_order"`
	IsDefault   bool   `json:"is_default"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

// Playlist 歌单
type Playlist struct {
	ID         string `json:"id"`
	IdentityID string `json:"identity_id"`
	Name       string `json:"name"`
	CoverURL   string `json:"cover_url,omitempty"`
	SortOrder  int    `json:"sort_order"`
	SongCount  int    `json:"song_count"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
}

// Song 歌曲索引
type Song struct {
	ID        string `json:"id"`
	Path      string `json:"path"`
	Source    string `json:"source"`
	Title     string `json:"title"`
	Artist    string `json:"artist,omitempty"`
	Album     string `json:"album,omitempty"`
	Duration  int    `json:"duration"`
	CoverURL  string `json:"cover_url,omitempty"`
	SortOrder int    `json:"sort_order,omitempty"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

// PlaylistSong 歌单-歌曲关联
type PlaylistSong struct {
	PlaylistID string `json:"playlist_id"`
	SongID     string `json:"song_id"`
	SortOrder  int    `json:"sort_order"`
	CreatedAt  int64  `json:"created_at"`
}

// PlaybackState 播放状态
type PlaybackState struct {
	IdentityID string `json:"identity_id"`
	PlaylistID string `json:"playlist_id,omitempty"`
	SongID     string `json:"song_id,omitempty"`
	Position   int    `json:"position"`
	Mode       string `json:"mode"`
	UpdatedAt  int64  `json:"updated_at"`
}

// SongProgress 单曲进度记忆
type SongProgress struct {
	IdentityID string `json:"identity_id"`
	SongID     string `json:"song_id"`
	Position   int    `json:"position"`
	UpdatedAt  int64  `json:"updated_at"`
}

// StorageSource 存储源
type StorageSource struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	Available bool   `json:"available"`
}

// DeviceLog 设备信息日志
type DeviceLog struct {
	ID             int64  `json:"id"`
	UserAgent      string `json:"user_agent"`
	ChromeVersion  int    `json:"chrome_version"`
	WebviewVersion int    `json:"webview_version"`
	IsWebview      bool   `json:"is_webview"`
	ScreenWidth    int    `json:"screen_width"`
	ScreenHeight   int    `json:"screen_height"`
	WindowWidth    int    `json:"window_width"`
	WindowHeight   int    `json:"window_height"`
	Language       string `json:"language"`
	Platform       string `json:"platform"`
	CookieEnabled  bool   `json:"cookie_enabled"`
	Online         bool   `json:"online"`
	Timestamp      string `json:"timestamp"`
	CreatedAt      int64  `json:"created_at"`
}

// APIResponse 统一响应
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ListResponse 列表响应
type ListResponse struct {
	Items interface{} `json:"items"`
	Total int         `json:"total"`
}
