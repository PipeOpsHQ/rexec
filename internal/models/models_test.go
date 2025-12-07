package models

import (
	"testing"
)

func TestTierLimits(t *testing.T) {
	trialLimits := GetTrialResourceLimits() // Get the values for the more generous tier

	tests := []struct {
		tier     string
		wantCPU  int64
		wantMem  int64
		wantDisk int64
	}{
		{"guest", GuestResourceLimits.CPUShares, GuestResourceLimits.MemoryMB, GuestResourceLimits.DiskMB},
		{"free", trialLimits.MaxCPUShares, trialLimits.MaxMemoryMB, trialLimits.MaxDiskMB},
		{"trial", trialLimits.MaxCPUShares, trialLimits.MaxMemoryMB, trialLimits.MaxDiskMB},
		{"pro", 2000, 2048, 10240},
		{"enterprise", 4000, 4096, 51200},
		{"unknown", trialLimits.MaxCPUShares, trialLimits.MaxMemoryMB, trialLimits.MaxDiskMB}, // Default to free/trial
	}

	for _, tt := range tests {
		t.Run(tt.tier, func(t *testing.T) {
			got := TierLimits(tt.tier)
			if got.CPUShares != tt.wantCPU {
				t.Errorf("TierLimits(%s).CPUShares = %d, want %d", tt.tier, got.CPUShares, tt.wantCPU)
			}
			if got.MemoryMB != tt.wantMem {
				t.Errorf("TierLimits(%s).MemoryMB = %d, want %d", tt.tier, got.MemoryMB, tt.wantMem)
			}
			if got.DiskMB != tt.wantDisk {
				t.Errorf("TierLimits(%s).DiskMB = %d, want %d", tt.tier, got.DiskMB, tt.wantDisk)
			}
		})
	}
}

func TestValidateTrialResources(t *testing.T) {
	customizationLimits := GetTrialResourceLimits()

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
			wantMem:  GuestResourceLimits.MemoryMB,
			wantCPU:  GuestResourceLimits.CPUShares,
			wantDisk: GuestResourceLimits.DiskMB,
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
			wantMem:  2048,
			wantCPU:  2000,
			wantDisk: 10240,
		},
		{
			name: "Free user with zero values (uses max trial defaults)",
			tier: "free",
			req: CreateContainerRequest{
				MemoryMB:  0,
				CPUShares: 0,
				DiskMB:    0,
			},
			// Expect Free/Trial defaults (max trial limits)
			wantMem:  customizationLimits.MaxMemoryMB,
			wantCPU:  customizationLimits.MaxCPUShares,
			wantDisk: customizationLimits.MaxDiskMB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateTrialResources(&tt.req, tt.tier)
			if got.MemoryMB != tt.wantMem {
				t.Errorf("ValidateTrialResources() MemoryMB = %d, want %d", got.MemoryMB, tt.wantMem)
			}
			if got.CPUShares != tt.wantCPU {
				t.Errorf("ValidateTrialResources() CPUShares = %d, want %d", got.CPUShares, tt.wantCPU)
			}
			if got.DiskMB != tt.wantDisk {
				t.Errorf("ValidateTrialResources() DiskMB = %d, want %d", got.DiskMB, tt.wantDisk)
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
