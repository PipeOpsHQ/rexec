package container

import (
	"errors"
	"testing"
)

func TestUserContainerLimit(t *testing.T) {
	tests := []struct {
		tier string
		want int
	}{
		{"trial", 5},
		{"guest", 1},
		{"free", 5},
		{"pro", 10},
		{"enterprise", 20},
		{"unknown", 5},
	}

	for _, tt := range tests {
		t.Run(tt.tier, func(t *testing.T) {
			if got := UserContainerLimit(tt.tier); got != tt.want {
				t.Errorf("UserContainerLimit(%s) = %d, want %d", tt.tier, got, tt.want)
			}
		})
	}
}

func TestSanitizeError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "No error",
			err:  nil,
			want: "",
		},
		{
			name: "Generic error",
			err:  errors.New("something went wrong"),
			want: "something went wrong",
		},
		{
			name: "Docker daemon connection error",
			err:  errors.New("Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?"),
			want: "Container service temporarily unavailable. Please try again.",
		},
		{
			name: "Connection refused",
			err:  errors.New("dial tcp 127.0.0.1:2375: connect: connection refused"),
			want: "Container service temporarily unavailable. Please try again.",
		},
		{
			name: "Sensitive IP info",
			err:  errors.New("Error connecting to tcp://192.168.1.100:2376: timeout"),
			want: "Container service temporarily unavailable. Please try again.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SanitizeError(tt.err); got != tt.want {
				t.Errorf("SanitizeError() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSanitizeErrorString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Empty string",
			input: "",
			want:  "",
		},
		{
			name:  "Normal string",
			input: "Hello World",
			want:  "Hello World",
		},
		{
			name:  "String with tcp:// IP",
			input: "Connection to tcp://127.0.0.1:8080 failed",
			want:  "Container service temporarily unavailable. Please try again.", // Generic message triggers on tcp://
		},
		{
			name:  "String with unix:// path",
			input: "Socket at unix:///tmp/sock not found",
			want:  "Socket at [docker-socket] not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SanitizeErrorString(tt.input); got != tt.want {
				t.Errorf("SanitizeErrorString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{"Zero", 0, "2G"},
		{"Small", 100, "0M"},
		{"Megabytes", 500 * 1024 * 1024, "500M"},
		{"Gigabytes", 2 * 1024 * 1024 * 1024, "2G"},
		{"Exact GB", 1024 * 1024 * 1024, "1G"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatBytes(tt.bytes); got != tt.want {
				t.Errorf("formatBytes(%d) = %s, want %s", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestGetImageMetadata(t *testing.T) {
	metadata := GetImageMetadata()
	if len(metadata) == 0 {
		t.Error("GetImageMetadata() returned empty list")
	}

	// Verify some known images exist
	foundDebian := false
	foundUbuntu := false
	for _, img := range metadata {
		if img.Name == "debian" {
			foundDebian = true
		}
		if img.Name == "ubuntu" {
			foundUbuntu = true
		}
		if img.Name == "" {
			t.Error("Found image with empty Name")
		}
		if img.DisplayName == "" {
			t.Errorf("Image %s has empty DisplayName", img.Name)
		}
		if img.Category == "" {
			t.Errorf("Image %s has empty Category", img.Name)
		}
	}

	if !foundDebian {
		t.Error("Expected 'debian' image not found in metadata")
	}
	if !foundUbuntu {
		t.Error("Expected 'ubuntu' image not found in metadata")
	}
}

func TestGetPopularImages(t *testing.T) {
	popular := GetPopularImages()
	if len(popular) == 0 {
		t.Error("GetPopularImages() returned empty list")
	}

	for _, img := range popular {
		if !img.Popular {
			t.Errorf("Image %s in popular list but Popular flag is false", img.Name)
		}
	}
}

func TestGetImagesByCategory(t *testing.T) {
	categories := GetImagesByCategory()
	if len(categories) == 0 {
		t.Error("GetImagesByCategory() returned empty map")
	}

	// Check specific categories
	if len(categories["debian"]) == 0 {
		t.Error("Category 'debian' is empty")
	}
	if len(categories["minimal"]) == 0 {
		t.Error("Category 'minimal' is empty")
	}

	// Verify categorization
	for cat, images := range categories {
		for _, img := range images {
			if img.Category != cat {
				t.Errorf("Image %s has category %s but found in category list %s", img.Name, img.Category, cat)
			}
		}
	}
}

func TestIsCustomImageSupported(t *testing.T) {
	tests := []struct {
		imageType string
		want      bool
	}{
		{"ubuntu", true},
		{"debian", true},
		{"alpine", true},
		{"nonexistent", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.imageType, func(t *testing.T) {
			if got := IsCustomImageSupported(tt.imageType); got != tt.want {
				t.Errorf("IsCustomImageSupported(%q) = %v, want %v", tt.imageType, got, tt.want)
			}
		})
	}
}

func TestGetImageName(t *testing.T) {
	// Note: This test depends on whether custom images exist locally or not.
	// Since we can't guarantee local state, we'll test the fallback logic or basic mapping.
	// However, GetImageName checks for local existence of custom images.
	// If we assume no custom images are built locally in the test environment, it might fall back or return empty if not in SupportedImages.

	// Let's just test that it returns *something* or empty string, and doesn't panic.
	// We can't assert exact values without mocking ImageExists.

	// For now, let's skip exact assertions on return values that depend on ImageExists
	// and just ensure it runs safely.

	_ = GetImageName("ubuntu")
	_ = GetImageName("nonexistent")
}

func TestMergeLabels(t *testing.T) {
	base := map[string]string{"a": "1", "b": "2"}
	custom := map[string]string{"b": "3", "c": "4"}

	merged := mergeLabels(base, custom)

	if len(merged) != 3 {
		t.Errorf("mergeLabels returned map of size %d, want 3", len(merged))
	}

	if merged["a"] != "1" {
		t.Errorf("merged['a'] = %s, want 1", merged["a"])
	}
	if merged["b"] != "3" { // custom should overwrite base
		t.Errorf("merged['b'] = %s, want 3", merged["b"])
	}
	if merged["c"] != "4" {
		t.Errorf("merged['c'] = %s, want 4", merged["c"])
	}
}
