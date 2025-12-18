package container

import (
	"strings"
	"testing"
)

func TestGenerateRoleScript(t *testing.T) {
	tests := []struct {
		name          string
		roleID        string
		wantErr       bool
		wantContains  []string
		checkVibeCoder bool
	}{
		{
			name:   "Standard Role",
			roleID: "standard",
			wantContains: []string{
				"Installing tools for role: The Minimalist",
				"unsetopt PROMPT_SP # Prevent partial line indicator (%)",
				"if [ ! -f /root/.zshrc ]; then",
				"if [ ! -f /home/user/.zshrc ]; then",
			},
		},
		{
			name:   "Node Role",
			roleID: "node",
			wantContains: []string{
				"Installing tools for role: 10x JS Ninja",
				"nodejs",
				"npm",
			},
		},
		{
			name:           "Vibe Coder Role",
			roleID:         "overemployed",
			checkVibeCoder: true,
			wantContains: []string{
				"Installing tools for role: Vibe Coder",
				"aider",
				"opencode",
			},
		},
		{
			name:    "Invalid Role",
			roleID:  "invalid_role_id",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script, err := GenerateRoleScript(tt.roleID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateRoleScript() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Check for expected strings
			for _, want := range tt.wantContains {
				if !strings.Contains(script, want) {
					t.Errorf("GenerateRoleScript() script missing expected string: %q", want)
				}
			}

			// Verify formatting of the Vibe Coder block specifically
			// This checks if the arguments to Sprintf were aligned correctly.
			// The original code had `if [ "%s" = "Vibe Coder" ]; then`
			// If Sprintf works correctly, it should be `if [ "Vibe Coder" = "Vibe Coder" ]; then` (or the role name inserted)
			if tt.checkVibeCoder {
				expectedCheck := `if [ "Vibe Coder" = "Vibe Coder" ]; then`
				if !strings.Contains(script, expectedCheck) {
					t.Errorf("GenerateRoleScript() script missing correctly formatted Vibe Coder check.\nExpected to find: %q\nThis implies fmt.Sprintf arguments are misaligned.", expectedCheck)
				}
			}
		})
	}
}

func TestAvailableRoles(t *testing.T) {
	roles := AvailableRoles()
	if len(roles) == 0 {
		t.Error("AvailableRoles() returned empty list")
	}

	// Verify required fields
	for _, role := range roles {
		if role.ID == "" {
			t.Error("Role ID cannot be empty")
		}
		if role.Name == "" {
			t.Errorf("Role Name cannot be empty for ID %s", role.ID)
		}
		// "minimal" role intentionally has no packages for fastest startup
		if len(role.Packages) == 0 && role.ID != "minimal" {
			t.Errorf("Role %s must have packages", role.ID)
		}
	}
}
