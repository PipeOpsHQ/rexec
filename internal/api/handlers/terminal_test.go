package handlers

import (
	"testing"
)

func TestIsValidUTF8Fast(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  bool
	}{
		{"Empty", []byte{}, true},
		{"ASCII", []byte("Hello World"), true},
		{"Valid UTF8", []byte("Hello 世界"), true},
		{"Invalid UTF8", []byte{0xff, 0xfe}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidUTF8Fast(tt.input); got != tt.want {
				t.Errorf("isValidUTF8Fast(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestFilterMouseTracking(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"No escape", "Hello World", "Hello World"},
		{"Standard escape", "\x1b[31mRed\x1b[0m", "\x1b[31mRed\x1b[0m"},
		{"Mouse press", "Click\x1b[<0;10;10MHere", "ClickHere"},
		{"Mouse release", "Release\x1b[<0;10;10mHere", "ReleaseHere"},
		{"Broken sequence", "\x1b[<0;10;", "\x1b[<0;10;"}, // Should preserve incomplete/broken
		{"Multiple sequences", "A\x1b[<0;1;1MB\x1b[<0;2;2mC", "ABC"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filterMouseTracking(tt.input); got != tt.want {
				t.Errorf("filterMouseTracking(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeUTF8(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{"Empty", []byte{}, ""},
		{"ASCII", []byte("Hello"), "Hello"},
		{"Valid UTF8", []byte("Hello 世界"), "Hello 世界"},
		{"Invalid start", []byte{0xff, 'H', 'i'}, "Hi"},
		{"Invalid middle", []byte{'H', 0xff, 'i'}, "Hi"},
		{"Invalid end", []byte{'H', 'i', 0xff}, "Hi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeUTF8(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeUTF8(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
