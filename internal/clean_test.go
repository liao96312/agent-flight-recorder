package afr

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPlanCleanSelectsOnlyExactCompletedSessions(t *testing.T) {
	root := t.TempDir()
	completed := completedCleanSession(t, root)
	active, err := NewSession(root, t.TempDir(), []string{"agent"}, testRedactor(t))
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanClean(root, CleanOptions{OlderThan: 24 * time.Hour, Now: time.Now().Add(48 * time.Hour)})
	if err == nil || !strings.Contains(err.Error(), active.Meta.ID) || len(plan.Targets) != 0 {
		t.Fatalf("active session was not refused: plan=%+v error=%v", plan, err)
	}
	if err := active.Finish("failed", nil, 0, ""); err != nil {
		t.Fatal(err)
	}
	plan, err = PlanClean(root, CleanOptions{OlderThan: 24 * time.Hour, Now: time.Now().Add(48 * time.Hour)})
	if err != nil || len(plan.Targets) != 2 || plan.Targets[0].ID != completed.Meta.ID || !filepath.IsAbs(plan.Targets[0].Path) {
		t.Fatalf("plan=%+v error=%v", plan, err)
	}
}

func TestPlanCleanRejectsForgedDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "forged"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := PlanClean(root, CleanOptions{OlderThan: time.Hour}); err == nil {
		t.Fatal("forged session directory accepted")
	}
}

func TestPlanCleanCapacityIsDeterministic(t *testing.T) {
	root := t.TempDir()
	first := completedCleanSession(t, root)
	time.Sleep(time.Millisecond)
	second := completedCleanSession(t, root)
	firstSize, _ := cleanSessionSize(first.Root)
	secondSize, _ := cleanSessionSize(second.Root)
	plan, err := PlanClean(root, CleanOptions{MaxBytes: secondSize})
	if err != nil || len(plan.Targets) != 1 || plan.Targets[0].ID != first.Meta.ID || plan.TotalBytes != firstSize {
		t.Fatalf("plan=%+v error=%v", plan, err)
	}
}

func TestPlanCleanSkipsResumableDesktopSession(t *testing.T) {
	for _, state := range []string{"active", "idle"} {
		for _, retention := range []string{"older-than", "max-bytes"} {
			t.Run(state+"/"+retention, func(t *testing.T) {
				root := t.TempDir()
				completed := completedCleanSession(t, root)
				desktop, err := NewSession(root, t.TempDir(), []string{"codex-desktop"}, testRedactor(t))
				if err != nil {
					t.Fatal(err)
				}
				desktop.Meta.CaptureMode = "desktop_hook"
				if err := desktop.Checkpoint(state, 0, ""); err != nil {
					t.Fatal(err)
				}
				options := CleanOptions{OlderThan: time.Hour, Now: time.Now().Add(48 * time.Hour)}
				if retention == "max-bytes" {
					options = CleanOptions{MaxBytes: 1}
					size, err := cleanSessionSize(desktop.Root)
					if err != nil {
						t.Fatal(err)
					}
					options.MaxBytes = size
				}
				plan, err := PlanClean(root, options)
				if err != nil || len(plan.Targets) != 1 || plan.Targets[0].ID != completed.Meta.ID {
					t.Fatalf("plan=%+v error=%v", plan, err)
				}
			})
		}
	}
}

func TestExecuteCleanRevalidatesThenDeletesExactTargets(t *testing.T) {
	root := t.TempDir()
	first := completedCleanSession(t, root)
	second := completedCleanSession(t, root)
	plan, err := PlanClean(root, CleanOptions{OlderThan: time.Hour, Now: time.Now().Add(48 * time.Hour)})
	if err != nil || len(plan.Targets) != 2 {
		t.Fatalf("plan=%+v error=%v", plan, err)
	}
	deleted, err := ExecuteClean(plan)
	if err != nil || len(deleted) != 2 {
		t.Fatalf("deleted=%+v error=%v", deleted, err)
	}
	for _, session := range []*Session{first, second} {
		if _, err := os.Stat(session.Root); !os.IsNotExist(err) {
			t.Fatalf("session still exists: %s", session.Root)
		}
	}
}

func TestExecuteCleanRefusesTargetChangedAfterPreview(t *testing.T) {
	root := t.TempDir()
	session := completedCleanSession(t, root)
	plan, err := PlanClean(root, CleanOptions{OlderThan: time.Hour, Now: time.Now().Add(48 * time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(session.Root, "changed"), []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ExecuteClean(plan); err == nil {
		t.Fatal("changed target was deleted")
	}
	if _, err := os.Stat(session.Root); err != nil {
		t.Fatalf("refused session was changed: %v", err)
	}
}

func completedCleanSession(t *testing.T, root string) *Session {
	t.Helper()
	session, err := NewSession(root, t.TempDir(), []string{"agent"}, testRedactor(t))
	if err != nil {
		t.Fatal(err)
	}
	exitCode := 0
	if err := session.Finish("completed", &exitCode, 0, ""); err != nil {
		t.Fatal(err)
	}
	return session
}
