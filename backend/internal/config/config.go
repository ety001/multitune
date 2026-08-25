package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config 应用配置
type Config struct {
	Port                    string
	DataPath                string
	CachePath               string
	DatabaseName            string
	MaxIdentities           int
	MaxPlaylistsPerIdentity int
	MaxSongsPerPlaylist     int
	ScanFormats             []string
	PlaybackSaveInterval    int
	LogLevel                string
	GINMode                 string
	StaticPath              string
	LazyCatDeploy           bool
	LazyCatUsername         string
	LazyCatDocumentRoot     string
	LazyCatMediaRoot        string
}

// Load 从环境变量加载配置
func Load() *Config {
	cfg := &Config{
		Port:                    getEnv("PORT", "8080"),
		DataPath:                getEnv("DATA_PATH", "/app/data"),
		CachePath:               getEnv("CACHE_PATH", ""),
		DatabaseName:            getEnv("DATABASE_NAME", "multitune.db"),
		MaxIdentities:           getEnvInt("MAX_IDENTITIES", 20),
		MaxPlaylistsPerIdentity: getEnvInt("MAX_PLAYLISTS_PER_IDENTITY", 50),
		MaxSongsPerPlaylist:     getEnvInt("MAX_SONGS_PER_PLAYLIST", 1000),
		ScanFormats:             getEnvSlice("SCAN_FORMATS", []string{"mp3", "flac", "m4a", "aac", "ogg", "wav"}),
		PlaybackSaveInterval:    getEnvInt("PLAYBACK_SAVE_INTERVAL", 5),
		LogLevel:                getEnv("LOG_LEVEL", "info"),
		GINMode:                 getEnv("GIN_MODE", "release"),
		StaticPath:              getEnv("STATIC_PATH", "/app/static"),
		LazyCatDeploy:           getEnvBool("LAZYCAT_DEPLOY", false),
		LazyCatUsername:         getEnv("LAZYCAT_USERNAME", ""),
		LazyCatDocumentRoot:     getEnv("LAZYCAT_DOCUMENT_ROOT", "/lzcapp/document"),
		LazyCatMediaRoot:        getEnv("LAZYCAT_MEDIA_ROOT", "/lzcapp/media"),
	}

	if _, err := strconv.Atoi(cfg.Port); err != nil {
		cfg.Port = "8080"
	}
	if cfg.MaxIdentities <= 0 {
		cfg.MaxIdentities = 20
	}
	if cfg.MaxPlaylistsPerIdentity <= 0 {
		cfg.MaxPlaylistsPerIdentity = 50
	}
	if cfg.MaxSongsPerPlaylist <= 0 {
		cfg.MaxSongsPerPlaylist = 1000
	}
	if cfg.PlaybackSaveInterval <= 0 {
		cfg.PlaybackSaveInterval = 5
	}
	// 缓存目录未显式配置时，落在数据目录下（懒猫部署应配置 CACHE_PATH
	// 指向 /lzcapp/cache 挂载，享受独立持久缓存区）
	if cfg.CachePath == "" {
		cfg.CachePath = filepath.Join(cfg.DataPath, "cache")
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultValue
	}
	return n
}

func getEnvBool(key string, defaultValue bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	switch strings.ToLower(v) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return defaultValue
	}
}

func getEnvSlice(key string, defaultValue []string) []string {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	parts := strings.Split(v, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
