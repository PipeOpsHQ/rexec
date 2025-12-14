package container

import (
	"strings"
	"testing"
)

func TestDefaultShellSetupConfig(t *testing.T) {
	cfg := DefaultShellSetupConfig()
	if !cfg.Enhanced {
		t.Error("Default config should have Enhanced=true")
	}
	if cfg.Theme != "robbyrussell" {
		t.Errorf("Default theme = %s, want robbyrussell", cfg.Theme)
	}
	if !cfg.GitAliases {
		t.Error("Default config should have GitAliases=true")
	}
}

func TestGenerateShellSetupScript(t *testing.T) {
	cfg := DefaultShellSetupConfig()
	script := generateShellSetupScript(cfg)

	// Check for formatting correction
	if !strings.Contains(script, "unsetopt PROMPT_SP # Prevent partial line indicator (%)") {
		t.Error("Script missing correctly formatted unsetopt PROMPT_SP line")
	}

	// Check for basic components
	wants := []string{
		"#!/bin/sh",
		"install_packages()",
		"install_ohmyzsh()",
		"ZSH_THEME=\"robbyrussell\"",
		"alias gs='git status'", // Git aliases
	}

	for _, want := range wants {
		if !strings.Contains(script, want) {
			t.Errorf("Script missing expected content: %q", want)
		}
	}
}

func TestGenerateShellSetupScript_Minimal(t *testing.T) {
	cfg := ShellSetupConfig{Enhanced: false}
	script := generateShellSetupScript(cfg)

	if !strings.Contains(script, "Minimal shell mode") {
		t.Error("Minimal script should contain minimal mode message")
	}
	if strings.Contains(script, "oh-my-zsh") {
		t.Error("Minimal script should not contain oh-my-zsh setup")
	}
}

func TestGeneratePluginInstallScript(t *testing.T) {
	cfg := ShellSetupConfig{
		Autosuggestions: true,
		SyntaxHighlight: true,
		HistorySearch:   true,
	}
	script := generatePluginInstallScript(cfg)

	wants := []string{
		"zsh-autosuggestions",
		"zsh-syntax-highlighting",
		"zsh-history-substring-search",
		"zsh-completions",
	}

	for _, want := range wants {
		if !strings.Contains(script, want) {
			t.Errorf("Plugin script missing: %q", want)
		}
	}
}

func TestGenerateShellSetupScript_Custom(t *testing.T) {
	cfg := ShellSetupConfig{
		Enhanced:    true,
		Theme:       "powerlevel10k",
		SystemStats: true,
	}
	script := generateShellSetupScript(cfg)

	if !strings.Contains(script, "ZSH_THEME=\"powerlevel10k\"") {
		t.Error("Script should contain custom theme")
	}
	if !strings.Contains(script, "show_system_stats()") {
		t.Error("Script should contain system stats function")
	}
	if !strings.Contains(script, "show_system_stats\n") {
		t.Error("Script should call show_system_stats")
	}
}
