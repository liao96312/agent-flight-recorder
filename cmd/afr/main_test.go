package main

import (
	"bytes"
	"errors"
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

func TestVersionIncludesBuildAndFormatContracts(t *testing.T) {
	value := versionString()
	for _, expected := range []string{"afr 0.1.0-dev", "commit=", "built=", "event_format=1", "manifest_format=1"} {
		if !strings.Contains(value, expected) {
			t.Fatalf("version missing %q: %s", expected, value)
		}
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

func TestHookFailsOpenOnBadInput(t *testing.T) {
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_ = write.Close()
	original := os.Stdin
	os.Stdin = read
	t.Cleanup(func() {
		os.Stdin = original
		_ = read.Close()
	})
	t.Setenv("PLUGIN_DATA", "")
	if code := hookCommand([]string{"--host", "codex"}); code != 0 {
		t.Fatalf("exit code = %d, want fail-open 0", code)
	}
}

func TestShowOptions(t *testing.T) {
	asJSON, openReport, selector, err := parseShowSelector([]string{"--open", "session-1"})
	if err != nil || asJSON || !openReport || selector != "session-1" {
		t.Fatalf("json=%t open=%t selector=%q error=%v", asJSON, openReport, selector, err)
	}
	if _, _, _, err := parseShowSelector([]string{"--json", "--open"}); err == nil {
		t.Fatal("show accepted --json with --open")
	}
}

func TestReportOpenCommandKeepsPathAsOneArgument(t *testing.T) {
	path := `C:\workspace with spaces\报告\report.html`
	command, args, err := reportOpenCommand("windows", path)
	if err != nil || command != "explorer.exe" || len(args) != 1 || args[0] != path {
		t.Fatalf("command=%q args=%q error=%v", command, args, err)
	}
}

func TestOpenReportPrintsAbsolutePathBeforeLaunchFailure(t *testing.T) {
	original := launchReport
	launchReport = func(string) error { return errors.New("failed") }
	t.Cleanup(func() { launchReport = original })

	var output bytes.Buffer
	path := filepath.Join(t.TempDir(), "report.html")
	if err := openReportFile(path, &output); err == nil || strings.TrimSpace(output.String()) != path {
		t.Fatalf("output=%q error=%v", output.String(), err)
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

func TestCleanWithoutYesIsPreviewOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	id := "20200101T000000.000000000Z-000000000000000000000000"
	root := filepath.Join(home, ".afr", "sessions", id)
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	metadata := `{"format_version":1,"id":"` + id + `","state":"completed","started_at":"2020-01-01T00:00:00Z","finished_at":"2020-01-01T00:00:01Z"}`
	if err := os.WriteFile(filepath.Join(root, "session.json"), []byte(metadata), 0o600); err != nil {
		t.Fatal(err)
	}
	if code := cleanCommand([]string{"--older-than", "1h"}); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("preview deleted session: %v", err)
	}
}
