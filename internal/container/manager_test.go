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
		{"guest", 5},
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
			name: "TCP connection error",
			err:  errors.New("dial tcp: lookup tcp://1.2.3.4:2376: no such host"),
			want: "Container service temporarily unavailable. Please try again.",
		},
		{
			name: "Error with IP address",
			err:  errors.New("failed to connect to tcp://192.168.1.1:2376"),
			want: "Container service temporarily unavailable. Please try again.",
		},
		{
			name: "Error with Unix socket path",
			err:  errors.New("dial unix:///var/run/docker.sock: connect: permission denied"),
			want: "dial [docker-socket] connect: permission denied", // Parser consumes the colon
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
