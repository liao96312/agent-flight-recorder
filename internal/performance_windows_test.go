package afr

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestLargeRepositoryPerformance(t *testing.T) {
	if os.Getenv("AFR_PERF") != "1" {
		t.Skip("set AFR_PERF=1 to run the fixed performance fixture")
	}

	const files, bytesPerFile, changedFiles = 10_000, 1024, 100
	root := t.TempDir()
	workspace, sessions := filepath.Join(root, "workspace"), filepath.Join(root, "sessions")
	if err := os.MkdirAll(workspace, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sessions, 0o700); err != nil {
		t.Fatal(err)
	}
	payload := bytes.Repeat([]byte("a"), bytesPerFile)
	for index := range files {
		directory := filepath.Join(workspace, formatIndex(index/100))
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, formatIndex(index%100)+".txt"), payload, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	git := requireGitRepository(t, workspace)
	fingerprinter, err := NewFingerprinter()
	if err != nil {
		t.Fatal(err)
	}
	redactor, err := NewRedactor(fingerprinter)
	if err != nil {
		t.Fatal(err)
	}
	limits := ScanLimits{MaxFiles: files + 1, MaxBytes: files * bytesPerFile * 2, MaxTime: 10 * time.Second}

	started := time.Now()
	before := CollectWorkspace(workspace, sessions, fingerprinter, limits)
	beforeDuration := time.Since(started)
	for index := range changedFiles {
		path := filepath.Join(workspace, formatIndex(index/100), formatIndex(index%100)+".txt")
		if err := os.WriteFile(path, bytes.Repeat([]byte("b"), bytesPerFile), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	started = time.Now()
	after := CollectWorkspace(workspace, sessions, fingerprinter, limits)
	afterDuration := time.Since(started)
	started = time.Now()
	delta := CompareWorkspace(before, after)
	deltaDuration := time.Since(started)
	started = time.Now()
	patch, err := GenerateWorkspacePatch(workspace, sessions, before, after, delta, fingerprinter, redactor, defaultDiffLimit)
	patchDuration := time.Since(started)

	t.Logf("fixture files=%d bytes=%d changed=%d git=%s", files, files*bytesPerFile, changedFiles, git)
	t.Logf("stages before=%s after=%s delta=%s patch=%s", beforeDuration, afterDuration, deltaDuration, patchDuration)
	if err != nil || before.Partial || after.Partial || len(delta.Modified) != changedFiles || patch.Error != "" || patch.Truncated {
		t.Fatalf("before.partial=%t after.partial=%t modified=%d patch=%+v error=%v", before.Partial, after.Partial, len(delta.Modified), patch, err)
	}
}

func requireGitRepository(t *testing.T, workspace string) string {
	t.Helper()
	git, err := exec.LookPath("git")
	if err != nil {
		t.Fatal("Git is required for the performance fixture")
	}
	for _, args := range [][]string{
		{"-C", workspace, "init", "-q"},
		{"-C", workspace, "config", "user.name", "AFR Fixture"},
		{"-C", workspace, "config", "user.email", "afr@example.invalid"},
		{"-C", workspace, "add", "."},
		{"-C", workspace, "commit", "-q", "-m", "fixture"},
	} {
		if output, err := exec.Command(git, args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
	output, err := exec.Command(git, "version").Output()
	if err != nil {
		t.Fatal(err)
	}
	return string(bytes.TrimSpace(output))
}

func formatIndex(value int) string {
	return string([]byte{'0' + byte(value/10), '0' + byte(value%10)})
}
