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

func TestVerifyRequiresOneSessionSelector(t *testing.T) {
	if code := verifyCommand(nil); code != exitUsage {
		t.Fatalf("exit code = %d, want %d", code, exitUsage)
	}
}
