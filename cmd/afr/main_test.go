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
