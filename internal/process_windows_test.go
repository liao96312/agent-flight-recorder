//go:build windows

package afr

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestHardKilledRecorderLeavesIncompleteSession(t *testing.T) {
	role := os.Getenv("AFR_TEST_HARD_ROLE")
	root := os.Getenv("AFR_TEST_HARD_ROOT")
	workspace := os.Getenv("AFR_TEST_HARD_WORKSPACE")
	marker := os.Getenv("AFR_TEST_HARD_MARKER")
	if role == "agent" {
		time.Sleep(time.Second)
		_ = os.WriteFile(marker, []byte("escaped"), 0o600)
		time.Sleep(30 * time.Second)
		os.Exit(0)
	}
	if role == "recorder" {
		_ = os.Setenv("AFR_TEST_HARD_ROLE", "agent")
		_, _ = Run(RunOptions{SessionsRoot: root, Workspace: workspace}, []string{os.Args[0], "-test.run=TestHardKilledRecorderLeavesIncompleteSession"})
		os.Exit(0)
	}

	directory := t.TempDir()
	root = filepath.Join(directory, "sessions")
	marker = filepath.Join(directory, "escaped.txt")
	command := exec.Command(os.Args[0], "-test.run=TestHardKilledRecorderLeavesIncompleteSession")
	command.Env = append(os.Environ(),
		"AFR_TEST_HARD_ROLE=recorder",
		"AFR_TEST_HARD_ROOT="+root,
		"AFR_TEST_HARD_WORKSPACE="+directory,
		"AFR_TEST_HARD_MARKER="+marker,
	)
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	finished := false
	defer func() {
		if !finished {
			_ = command.Process.Kill()
			_ = command.Wait()
		}
	}()
	deadline := time.Now().Add(3 * time.Second)
	ready := false
	for time.Now().Before(deadline) {
		entries, _ := os.ReadDir(root)
		if len(entries) == 1 {
			events, _ := os.ReadFile(filepath.Join(root, entries[0].Name(), "events.jsonl"))
			if bytes.Contains(events, []byte(`"type":"process_started"`)) {
				ready = true
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !ready {
		t.Fatal("recorder did not create evidence before deadline")
	}
	if err := command.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = command.Wait()
	finished = true
	time.Sleep(1200 * time.Millisecond)
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("agent escaped after recorder hard kill: %v", err)
	}
	summaries, err := ListSessions(root)
	if err != nil || len(summaries) != 1 || !summaries[0].Incomplete || summaries[0].State != "incomplete" {
		t.Fatalf("summaries=%+v error=%v", summaries, err)
	}
}
