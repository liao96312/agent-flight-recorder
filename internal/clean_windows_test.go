package afr

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestPlanCleanRejectsJunctionArtifact(t *testing.T) {
	root := t.TempDir()
	session := completedCleanSession(t, root)
	target := t.TempDir()
	link := filepath.Join(session.Root, "linked")
	if output, err := exec.Command("cmd.exe", "/d", "/c", "mklink", "/J", link, target).CombinedOutput(); err != nil {
		t.Fatalf("create junction: %v: %s", err, output)
	}
	defer os.Remove(link)
	if _, err := PlanClean(root, CleanOptions{OlderThan: time.Hour, Now: time.Now().Add(48 * time.Hour)}); err == nil {
		t.Fatal("junction artifact accepted")
	}
}

func TestPlanCleanRejectsJunctionSessionEntry(t *testing.T) {
	root := t.TempDir()
	target := t.TempDir()
	link := filepath.Join(root, "20260721T000000.000000000Z-aaaaaaaaaaaaaaaaaaaaaaaa")
	if output, err := exec.Command("cmd.exe", "/d", "/c", "mklink", "/J", link, target).CombinedOutput(); err != nil {
		t.Fatalf("create junction: %v: %s", err, output)
	}
	defer os.Remove(link)
	if _, err := PlanClean(root, CleanOptions{OlderThan: time.Hour}); err == nil {
		t.Fatal("junction session entry accepted")
	}
}
