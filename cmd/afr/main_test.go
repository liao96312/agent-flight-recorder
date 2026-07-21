package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestShowRejectsUnimplementedOpenOption(t *testing.T) {
	if code := showCommand([]string{"--open"}); code != exitUsage {
		t.Fatalf("exit code = %d, want %d", code, exitUsage)
	}
}

func TestCleanValueParsers(t *testing.T) {
	if duration, err := parseCleanDuration("30d"); err != nil || duration != 30*24*time.Hour {
		t.Fatalf("duration=%s error=%v", duration, err)
	}
	if size, err := parseByteSize("2GiB"); err != nil || size != 2<<30 {
		t.Fatalf("size=%d error=%v", size, err)
	}
	if _, err := parseByteSize("0"); err == nil {
		t.Fatal("zero byte limit accepted")
	}
}

func TestCleanConfirmationDefaultsToNo(t *testing.T) {
	for _, test := range []struct {
		input string
		want  bool
	}{{"\n", false}, {"no\n", false}, {"yes\n", true}, {"Y\n", true}} {
		var output bytes.Buffer
		got, err := confirmClean(strings.NewReader(test.input), &output)
		if err != nil || got != test.want || output.String() == "" {
			t.Fatalf("input=%q got=%t want=%t output=%q error=%v", test.input, got, test.want, output.String(), err)
		}
	}
}
