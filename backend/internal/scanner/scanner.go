package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ety001/multitune/internal/fsutil"
	"github.com/ety001/multitune/internal/model"
	"github.com/ety001/multitune/internal/repository"
)

// Scanner 歌曲扫描器
type Scanner struct {
	repo     *repository.SongRepo
	mu       sync.Mutex
	scanning bool
}

// New 创建扫描器
func New(repo *repository.SongRepo, formats []string) *Scanner {
	// formats 目前由 fsutil.IsAudioFile 统一维护，后续可按需接入 SCAN_FORMATS 配置
	_ = formats
	return &Scanner{
		repo: repo,
	}
}

// ErrScanInProgress 扫描任务正在进行中
var ErrScanInProgress = fmt.Errorf("扫描任务正在进行中")

// ScanProgressFunc 扫描进度回调
type ScanProgressFunc func(current, total int)

// ScanResult 扫描结果
type ScanResult struct {
	Scanned int          `json:"scanned"`
	Added   int          `json:"added"`
	Updated int          `json:"updated"`
	Songs   []model.Song `json:"songs"`
}

// ScanPath 扫描指定路径（文件或目录）
func (s *Scanner) ScanPath(path string) (*ScanResult, error) {
	return s.ScanPaths([]string{path}, nil)
}

// ScanPaths 批量扫描多个路径，支持进度回调
// progress 回调参数：current 为已处理文件数（含非音频文件），total 为文件总数
func (s *Scanner) ScanPaths(paths []string, progress ScanProgressFunc) (*ScanResult, error) {
	s.mu.Lock()
	if s.scanning {
		s.mu.Unlock()
		return nil, ErrScanInProgress
	}
	s.scanning = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.scanning = false
		s.mu.Unlock()
	}()

	result := &ScanResult{
		Songs: []model.Song{},
	}

	// 第一阶段：统计所有待处理文件总数（包含非音频文件）
	if progress != nil {
		progress(0, 0)
	}

	total := 0
	for _, path := range paths {
		count, err := s.countFiles(path)
		if err != nil {
			return nil, err
		}
		total += count
	}

	// 第二阶段：逐个文件处理
	current := 0
	for _, path := range paths {
		if err := s.scanPathWithProgress(path, result, &current, total, progress); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// countFiles 统计路径下所有文件数量（包含非音频文件）
func (s *Scanner) countFiles(path string) (int, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("获取路径信息失败: %w", err)
	}

	if !info.IsDir() {
		return 1, nil
	}

	count := 0
	err = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			slog.Warn("统计路径失败", "path", path, "error", err)
			return nil
		}
		if !info.IsDir() {
			count++
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}

// scanPathWithProgress 扫描单个路径并更新进度
func (s *Scanner) scanPathWithProgress(path string, result *ScanResult, current *int, total int, progress ScanProgressFunc) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("获取路径信息失败: %w", err)
	}

	if !info.IsDir() {
		*current++
		if progress != nil {
			progress(*current, total)
		}
		if !fsutil.IsAudioFile(path) {
			return nil
		}
		return s.scanFile(path, result)
	}

	return filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			slog.Warn("扫描路径失败", "path", filePath, "error", err)
			return nil
		}
		if info.IsDir() {
			return nil
		}

		*current++
		if progress != nil {
			progress(*current, total)
		}

		if !fsutil.IsAudioFile(filePath) {
			return nil
		}
		return s.scanFile(filePath, result)
	})
}

// scanFile 扫描单个文件
func (s *Scanner) scanFile(path string, result *ScanResult) error {
	result.Scanned++

	source := s.inferSource(path)
	title, artist, album, duration := s.readMetadata(path)

	upsertResult, err := s.repo.Upsert(path, source, title, artist, album, duration)
	if err != nil {
		return fmt.Errorf("保存歌曲失败 %s: %w", path, err)
	}

	if upsertResult.IsNew {
		result.Added++
	} else {
		result.Updated++
	}
	result.Songs = append(result.Songs, *upsertResult.Song)

	return nil
}

// inferSource 根据路径推断来源
// 取绝对路径的第一级目录名作为 source，例如 /home/user/music/xxx.mp3 -> home。
// 无法推断时返回 unknown。
func (s *Scanner) inferSource(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "unknown"
	}
	parts := strings.Split(abs, string(filepath.Separator))
	for _, p := range parts {
		if p != "" {
			return p
		}
	}
	return "unknown"
}

// readMetadata 读取音频元数据
func (s *Scanner) readMetadata(path string) (title, artist, album string, duration int) {
	title = fallbackTitle(path)

	// 尝试使用 ffprobe
	if metadata, err := s.readWithFFProbe(path); err == nil {
		if metadata.Format.Tags != nil {
			tags := metadata.Format.Tags
			if tags.Title != "" {
				title = tags.Title
			}
			if tags.Artist != "" {
				artist = tags.Artist
			}
			if tags.Album != "" {
				album = tags.Album
			}
		}
		if metadata.Format.Duration > 0 {
			duration = int(metadata.Format.Duration + 0.5)
		}
	} else {
		slog.Debug("ffprobe 读取失败，使用文件名作为标题", "path", path, "error", err)
	}

	return
}

// fallbackTitle 从文件名提取标题
func fallbackTitle(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

// ffprobeMetadata ffprobe JSON 输出结构
type ffprobeMetadata struct {
	Format struct {
		Duration float64 `json:"duration"`
		Tags     *struct {
			Title  string `json:"title"`
			Artist string `json:"artist"`
			Album  string `json:"album"`
		} `json:"tags"`
	} `json:"format"`
}

// readWithFFProbe 使用 ffprobe 读取元数据
func (s *Scanner) readWithFFProbe(path string) (*ffprobeMetadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-show_format",
		"-of", "json",
		path,
	)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe 执行失败: %w", err)
	}

	var metadata ffprobeMetadata
	if err := json.Unmarshal(output, &metadata); err != nil {
		return nil, fmt.Errorf("解析 ffprobe 输出失败: %w", err)
	}

	return &metadata, nil
}
