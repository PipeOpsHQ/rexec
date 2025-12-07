package models

import (
	"testing"
	"time"
)

func TestTierLimits(t *testing.T) {
	tests := []struct {
		name              string
		tier              string
		subActive         bool
		wantCPU           int64
		wantMem           int64
		wantDisk          int64
		wantSessionLimit  time.Duration
	}{
		{"Guest", "guest", false, 500, 512, 2048, 1 * time.Hour},
		{"Free (No Sub)", "free", false, 2000, 2048, 10240, 50 * time.Hour},
		{"Free (With Sub)", "free", true, 4000, 4096, 20480, 0},
		{"Pro (Legacy)", "pro", false, 4000, 4096, 20480, 0},
		{"Enterprise", "enterprise", false, 8000, 8192, 51200, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetUserResourceLimits(tt.tier, tt.subActive)
			if got.CPUShares != tt.wantCPU {
				t.Errorf("CPUShares = %d, want %d", got.CPUShares, tt.wantCPU)
			}
			if got.MemoryMB != tt.wantMem {
				t.Errorf("MemoryMB = %d, want %d", got.MemoryMB, tt.wantMem)
			}
			if got.DiskMB != tt.wantDisk {
				t.Errorf("DiskMB = %d, want %d", got.DiskMB, tt.wantDisk)
			}
			if got.SessionDuration != tt.wantSessionLimit {
				t.Errorf("SessionDuration = %v, want %v", got.SessionDuration, tt.wantSessionLimit)
			}
		})
	}
}

func TestValidateTrialResources(t *testing.T) {
	customizationLimits := GetTrialResourceLimits()
	// Default Free limits (No Sub)
	freeDefaults := GetUserResourceLimits("free", false)

	tests := []struct {
		name      string
		tier      string
		req       CreateContainerRequest
		wantCPU   int64
		wantMem   int64
		wantDisk  int64
	}{
		{
			name: "Guest user with custom request (ignored, uses fixed guest limits)",
			tier: "guest",
			req: CreateContainerRequest{
				MemoryMB:  1024,
				CPUShares: 1000,
				DiskMB:    2048,
			},
			wantMem:  512,
			wantCPU:  500,
			wantDisk: 2048,
		},
		{
			name: "Free user within customization limits",
			tier: "free",
			req: CreateContainerRequest{
				MemoryMB:  1024,
				CPUShares: 1000,
				DiskMB:    2048,
			},
			wantMem:  1024,
			wantCPU:  1000,
			wantDisk: 2048,
		},
		{
			name: "Free user exceeds max customization limits (clamped)",
			tier: "free",
			req: CreateContainerRequest{
				MemoryMB:  customizationLimits.MaxMemoryMB + 100,
				CPUShares: customizationLimits.MaxCPUShares + 100,
				DiskMB:    customizationLimits.MaxDiskMB + 100,
			},
			// Clamped to max allowed customization
			wantMem:  customizationLimits.MaxMemoryMB,
			wantCPU:  customizationLimits.MaxCPUShares,
			wantDisk: customizationLimits.MaxDiskMB,
		},
		{
			name: "Free user below min customization limits (clamped)",
			tier: "free",
			req: CreateContainerRequest{
				MemoryMB:  customizationLimits.MinMemoryMB - 100,
				CPUShares: customizationLimits.MinCPUShares - 100,
				DiskMB:    customizationLimits.MinDiskMB - 100,
			},
			// Clamped to min allowed customization
			wantMem:  customizationLimits.MinMemoryMB,
			wantCPU:  customizationLimits.MinCPUShares,
			wantDisk: customizationLimits.MinDiskMB,
		},
		{
			name: "Pro user ignores custom request (uses fixed pro limits)",
			tier: "pro",
			req: CreateContainerRequest{
				MemoryMB:  512,
				CPUShares: 500,
				DiskMB:    1024,
			},
			// Expect Pro defaults
			wantMem:  4096, // Pro is 4GB
			wantCPU:  4000, // Pro is 4 vCPU
			wantDisk: 20480, // Pro is 20GB
		},
		{
			name: "Free user with zero values (uses Free tier defaults)",
			tier: "free",
			req: CreateContainerRequest{
				MemoryMB:  0,
				CPUShares: 0,
				DiskMB:    0,
			},
			// Expect Free defaults (No Sub)
			wantMem:  freeDefaults.MemoryMB,
			wantCPU:  freeDefaults.CPUShares,
			wantDisk: freeDefaults.DiskMB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateTrialResources(&tt.req, tt.tier)
			if got.MemoryMB != tt.wantMem {
				t.Errorf("MemoryMB = %d, want %d", got.MemoryMB, tt.wantMem)
			}
			if got.CPUShares != tt.wantCPU {
				t.Errorf("CPUShares = %d, want %d", got.CPUShares, tt.wantCPU)
			}
			if got.DiskMB != tt.wantDisk {
				t.Errorf("DiskMB = %d, want %d", got.DiskMB, tt.wantDisk)
			}
		})
	}
}

func TestDefaultShellConfig(t *testing.T) {
	cfg := DefaultShellConfig()
	if !cfg.Enhanced {
		t.Error("Default shell should be enhanced")
	}
	if cfg.Theme != "rexec" {
		t.Errorf("Default theme should be rexec, got %s", cfg.Theme)
	}
}

func TestMinimalShellConfig(t *testing.T) {
	cfg := MinimalShellConfig()
	if cfg.Enhanced {
		t.Error("Minimal shell should not be enhanced")
	}
	if cfg.Theme != "" {
		t.Errorf("Minimal shell theme should be empty, got %s", cfg.Theme)
	}
}