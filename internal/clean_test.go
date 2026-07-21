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
