package settings

import "testing"

// TestSudoUserNameValidation guards the SUDO_USER path-traversal fix: crafted
// names must never reach the filesystem lookup.
func TestSudoUserNameValidation(t *testing.T) {
	t.Setenv("SUDO_USER", "../../etc")
	if p, ok := sudoUserSettingsPath(); ok {
		t.Fatalf("traversal name must be rejected, got %q", p)
	}
	t.Setenv("SUDO_USER", "root")
	if _, ok := sudoUserSettingsPath(); ok {
		t.Fatal("root must be rejected")
	}
	t.Setenv("SUDO_USER", "no such user with spaces")
	if _, ok := sudoUserSettingsPath(); ok {
		t.Fatal("invalid name must be rejected")
	}
}
