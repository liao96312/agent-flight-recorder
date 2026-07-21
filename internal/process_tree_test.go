package afr

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestProcessTreeKillsGrandchild(t *testing.T) {
	role := os.Getenv("AFR_TEST_TREE_ROLE")
	marker := os.Getenv("AFR_TEST_TREE_MARKER")
	spawned := os.Getenv("AFR_TEST_TREE_SPAWNED")
	if role == "grandchild" {
		time.Sleep(750 * time.Millisecond)
		_ = os.WriteFile(marker, []byte("escaped"), 0o600)
		os.Exit(0)
	}
	if role == "child" {
		time.Sleep(100 * time.Millisecond)
		_ = os.Setenv("AFR_TEST_TREE_ROLE", "grandchild")
		command := exec.Command(os.Args[0], "-test.run=TestProcessTreeKillsGrandchild")
		if err := command.Start(); err != nil {
			os.Exit(2)
		}
		_ = os.WriteFile(spawned, []byte("spawned"), 0o600)
		time.Sleep(30 * time.Second)
		os.Exit(0)
	}

	directory := t.TempDir()
	marker = filepath.Join(directory, "escaped.txt")
	spawned = filepath.Join(directory, "spawned.txt")
	t.Setenv("AFR_TEST_TREE_ROLE", "child")
	t.Setenv("AFR_TEST_TREE_MARKER", marker)
	t.Setenv("AFR_TEST_TREE_SPAWNED", spawned)
	interrupts := make(chan os.Signal, 2)
	go func() {
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := os.Stat(spawned); err == nil {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		interrupts <- os.Interrupt
		interrupts <- os.Interrupt
	}()
	result, err := Run(RunOptions{
		SessionsRoot: filepath.Join(directory, "sessions"),
		Workspace:    directory,
		Interrupts:   interrupts,
		GracePeriod:  time.Hour,
	}, []string{os.Args[0], "-test.run=TestProcessTreeKillsGrandchild"})
	if err != nil {
		t.Fatalf("result=%+v error=%v", result, err)
	}
	time.Sleep(time.Second)
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("grandchild escaped process tree: %v", err)
	}
}
