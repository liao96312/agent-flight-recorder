//go:build windows

package afr

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestReleaseHundredForcedTerminationsHaveValidEvidence(t *testing.T) {
	if os.Getenv("AFR_RELEASE") != "1" {
		t.Skip("set AFR_RELEASE=1 to run release stress tests")
	}
	if os.Getenv("AFR_RELEASE_FORCE_CHILD") == "1" {
		signal.Ignore(os.Interrupt)
		time.Sleep(30 * time.Second)
		os.Exit(0)
	}

	root := t.TempDir()
	workspace := t.TempDir()
	t.Setenv("AFR_RELEASE_FORCE_CHILD", "1")
	for iteration := 1; iteration <= 100; iteration++ {
		interrupts := make(chan os.Signal, 2)
		interrupts <- os.Interrupt
		interrupts <- os.Interrupt
		result, err := Run(RunOptions{
			SessionsRoot: root,
			Workspace:    workspace,
			Interrupts:   interrupts,
			GracePeriod:  time.Hour,
		}, []string{os.Args[0], "-test.run=TestReleaseHundredForcedTerminationsHaveValidEvidence"})
		if err != nil {
			t.Fatalf("iteration %d: %v", iteration, err)
		}
		verified := VerifySession(result.SessionDir)
		if !verified.Valid {
			t.Fatalf("iteration %d: invalid evidence: %+v", iteration, verified)
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 100 {
		t.Fatalf("sessions=%d error=%v", len(entries), err)
	}
}

func TestReleaseHundredConcurrentOutputWriters(t *testing.T) {
	if os.Getenv("AFR_RELEASE") != "1" {
		t.Skip("set AFR_RELEASE=1 to run release stress tests")
	}
	if os.Getenv("AFR_RELEASE_OUTPUT_CHILD") == "1" {
		var writers sync.WaitGroup
		writers.Add(100)
		for writer := range 100 {
			go func() {
				defer writers.Done()
				for line := range 100 {
					stream := os.Stdout
					if writer%2 == 1 {
						stream = os.Stderr
					}
					_, _ = fmt.Fprintf(stream, "writer=%03d line=%03d\n", writer, line)
				}
			}()
		}
		writers.Wait()
		os.Exit(0)
	}

	t.Setenv("AFR_RELEASE_OUTPUT_CHILD", "1")
	result, err := Run(RunOptions{
		SessionsRoot: filepath.Join(t.TempDir(), "sessions"),
		Workspace:    t.TempDir(),
	}, []string{os.Args[0], "-test.run=TestReleaseHundredConcurrentOutputWriters"})
	if err != nil || result.ExitCode != 0 {
		t.Fatalf("result=%+v error=%v", result, err)
	}
	verified := VerifySession(result.SessionDir)
	if !verified.Valid || verified.Events < 10_006 {
		t.Fatalf("concurrent output evidence is incomplete: %+v", verified)
	}
}
