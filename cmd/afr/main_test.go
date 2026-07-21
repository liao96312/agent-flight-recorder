package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUsageErrorCreatesNoSession(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	if code := realMain(nil); code != exitUsage {
		t.Fatalf("exit code = %d, want %d", code, exitUsage)
	}
	if _, err := os.Stat(filepath.Join(home, ".afr")); !os.IsNotExist(err) {
		t.Fatalf("usage error created session storage: %v", err)
	}
}

func TestVerifyRejectsMultipleSessionSelectors(t *testing.T) {
	if code := verifyCommand([]string{"one", "two"}); code != exitUsage {
		t.Fatalf("exit code = %d, want %d", code, exitUsage)
	}
}

func TestRunRequiresArgvSeparator(t *testing.T) {
	if code := runCommand([]string{"cmd.exe", "/c", "echo hello"}); code != exitUsage {
		t.Fatalf("exit code = %d, want %d", code, exitUsage)
	}
}
