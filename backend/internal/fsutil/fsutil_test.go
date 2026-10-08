package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsAudioFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"song.mp3", true},
		{"song.MP3", true},
		{"song.flac", true},
		{"song.m4a", true},
		{"song.txt", false},
		{"song", false},
	}
	for _, tt := range tests {
		if got := IsAudioFile(tt.path); got != tt.want {
			t.Errorf("IsAudioFile(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestListDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "song.mp3"), []byte("dummy"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0755); err != nil {
		t.Fatal(err)
	}

	items, err := ListDirectory(dir)
	if err != nil {
		t.Fatalf("ListDirectory failed: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("item count = %d, want 2", len(items))
	}

	var foundAudio bool
	for _, item := range items {
		if item.Name == "song.mp3" {
			if !item.IsAudio {
				t.Error("song.mp3 should be marked as audio")
			}
			foundAudio = true
		}
	}
	if !foundAudio {
		t.Error("audio file not found in listing")
	}
}

func TestListDirectoryNaturalOrder(t *testing.T) {
	dir := t.TempDir()
	names := []string{
		"10.冰雪奇缘·10·向北山出发.mp3",
		"11.冰雪奇缘·11·到达北山.mp3",
		"12.冰雪奇缘·12·再次受伤的安娜.mp3",
		"1.【试听】冰雪奇缘·01.mp3",
		"2.【试听】冰雪奇缘·02·安娜得救了.mp3",
		"3.冰雪奇缘·03·寂静的城堡.mp3",
		"9.冰雪奇缘·09.mp3",
	}
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("dummy"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "子目录"), 0755); err != nil {
		t.Fatal(err)
	}

	items, err := ListDirectory(dir)
	if err != nil {
		t.Fatalf("ListDirectory failed: %v", err)
	}

	got := make([]string, 0, len(items))
	for _, item := range items {
		got = append(got, item.Name)
	}
	want := []string{
		"1.【试听】冰雪奇缘·01.mp3",
		"2.【试听】冰雪奇缘·02·安娜得救了.mp3",
		"3.冰雪奇缘·03·寂静的城堡.mp3",
		"9.冰雪奇缘·09.mp3",
		"10.冰雪奇缘·10·向北山出发.mp3",
		"11.冰雪奇缘·11·到达北山.mp3",
		"12.冰雪奇缘·12·再次受伤的安娜.mp3",
		"子目录",
	}

	if len(got) != len(want) {
		t.Fatalf("条目数 = %d, want %d, got = %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("排序结果不符:\n got = %v\nwant = %v", got, want)
		}
	}
}

func TestIsPathAllowed(t *testing.T) {
	roots := []string{"/lzcapp/media", "/lzcapp/document/alice"}

	cases := []struct {
		name string
		path string
		want bool
	}{
		{"根自身-媒体", "/lzcapp/media", true},
		{"根自身-用户文档", "/lzcapp/document/alice", true},
		{"根下子目录", "/lzcapp/media/music", true},
		{"用户文档下子目录", "/lzcapp/document/alice/music", true},
		{"尾部斜杠", "/lzcapp/media/", true},
		{"冗余斜杠", "/lzcapp//media/./music", true},
		{"前缀相似不可绕过", "/lzcapp/media2", false},
		{"前缀相似目录不可绕过", "/lzcapp/document/alice2", false},
		{"其他用户目录", "/lzcapp/document/bob", false},
		{"文档总根", "/lzcapp/document", false},
		{"上级目录", "/lzcapp", false},
		{"根路径", "/", false},
		{"完全无关路径", "/etc/passwd", false},
		{"路径穿越归一化后拒绝", "/lzcapp/document/alice/../../bob/music", false},
		{"路径穿越逃逸到根", "/lzcapp/media/..", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsPathAllowed(tc.path, roots); got != tc.want {
				t.Errorf("IsPathAllowed(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

func TestValidateUsername(t *testing.T) {
	valid := []string{"alice", "bob_01", "user.name", "a-b-c", "ety001"}
	invalid := []string{"", ".", "..", "a/b", "../etc", "a\\b", "a b", "a;b", "用户"}

	for _, u := range valid {
		if !ValidateUsername(u) {
			t.Errorf("ValidateUsername(%q) 应为 true", u)
		}
	}
	for _, u := range invalid {
		if ValidateUsername(u) {
			t.Errorf("ValidateUsername(%q) 应为 false", u)
		}
	}
}
