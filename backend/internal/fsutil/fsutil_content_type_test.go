package fsutil

import (
	"testing"
)

func TestContentTypeByExt(t *testing.T) {
	tests := []struct {
		ext  string
		want string
	}{
		{".mp3", "audio/mpeg"},
		{".flac", "audio/flac"},
		{".m4a", "audio/mp4"},
		{".aac", "audio/aac"},
		{".ogg", "audio/ogg"},
		{".wav", "audio/wav"},
		{".MP3", "audio/mpeg"},
		{".txt", "application/octet-stream"},
		{"", "application/octet-stream"},
	}
	for _, tt := range tests {
		if got := ContentTypeByExt(tt.ext); got != tt.want {
			t.Errorf("ContentTypeByExt(%q) = %q, want %q", tt.ext, got, tt.want)
		}
	}
}
