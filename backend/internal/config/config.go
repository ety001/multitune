package config

import (
	"os"
	"strconv"
	"strings"
)

// Config 应用配置
type Config struct {
	Port                   string
	DataPath               string
	MediaRoot              string
	DatabaseName           string
	MaxIdentities          int
	MaxPlaylistsPerIdentity int
	MaxSongsPerPlaylist    int
	ScanFormats            []string
	PlaybackSaveInterval   int
	LogLevel               string
}

// Load 从环境变量加载配置
func Load() *Config {
	return &Config{
		Port:                    getEnv("PORT", "8080"),
		DataPath:                getEnv("DATA_PATH", "/app/data"),
		MediaRoot:               getEnv("MEDIA_ROOT", "/app/media"),
		DatabaseName:            getEnv("DATABASE_NAME", "multitune.db"),
		MaxIdentities:           getEnvInt("MAX_IDENTITIES", 20),
		MaxPlaylistsPerIdentity: getEnvInt("MAX_PLAYLISTS_PER_IDENTITY", 50),
		MaxSongsPerPlaylist:     getEnvInt("MAX_SONGS_PER_PLAYLIST", 1000),
		ScanFormats:             getEnvSlice("SCAN_FORMATS", []string{"mp3", "flac", "m4a", "aac", "ogg", "wav"}),
		PlaybackSaveInterval:    getEnvInt("PLAYBACK_SAVE_INTERVAL", 5),
		LogLevel:                getEnv("LOG_LEVEL", "info"),
	}
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
