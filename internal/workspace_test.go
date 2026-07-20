package afr

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestWorkspaceScanAndDelta(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "keep.txt"), []byte("before"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "rename.txt"), []byte("same"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "delete.txt"), []byte("gone"), 0o600); err != nil {
		t.Fatal(err)
	}
	fingerprinter, err := NewFingerprinter()
	if err != nil {
		t.Fatal(err)
	}
	before := CollectWorkspace(root, "", fingerprinter, ScanLimits{})
	if err := os.WriteFile(filepath.Join(root, "keep.txt"), []byte("after"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(root, "rename.txt"), filepath.Join(root, "renamed.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "added.txt"), []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "delete.txt")); err != nil {
		t.Fatal(err)
	}
	after := CollectWorkspace(root, "", fingerprinter, ScanLimits{})
	delta := CompareWorkspace(before, after)
	if !reflect.DeepEqual(delta.Added, []string{"added.txt"}) || !reflect.DeepEqual(delta.Modified, []string{"keep.txt"}) || !reflect.DeepEqual(delta.Deleted, []string{"delete.txt"}) || !reflect.DeepEqual(delta.Renamed, []RenameEvidence{{From: "rename.txt", To: "renamed.txt"}}) {
		t.Fatalf("delta=%+v", delta)
	}
}

func TestSessionFingerprintsDoNotLinkAcrossSessions(t *testing.T) {
	first, err := NewFingerprinter()
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewFingerprinter()
	if err != nil {
		t.Fatal(err)
	}
	data := []byte("same low entropy value")
	if first.Bytes(data) != first.Bytes(data) || first.Bytes(data) == second.Bytes(data) {
		t.Fatal("session fingerprint correlation boundary failed")
	}
}

func TestResolveWorkspaceRelativePath(t *testing.T) {
	parent := t.TempDir()
	workspace := filepath.Join(parent, "workspace")
	if err := os.Mkdir(workspace, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(parent)
	resolved, err := resolveWorkspace("workspace")
	if err != nil {
		t.Fatal(err)
	}
	expected, _ := filepath.EvalSymlinks(workspace)
	if resolved != expected {
		t.Fatalf("resolved=%q want=%q", resolved, expected)
	}
}

func TestWorkspaceScanBudgetIsPartial(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"a", "b"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("1234"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	fingerprinter, _ := NewFingerprinter()
	snapshot := CollectWorkspace(root, "", fingerprinter, ScanLimits{MaxFiles: 1, MaxBytes: 4, MaxTime: time.Second})
	if !snapshot.Partial || snapshot.ScannedFiles != 1 {
		t.Fatalf("snapshot=%+v", snapshot)
	}
}

func TestParsePorcelainV2(t *testing.T) {
	input := []byte("# branch.oid abc\x001 .M N... 100644 100644 100644 aaa aaa file.txt\x002 R. N... 100644 100644 100644 aaa bbb R100 new.txt\x00old.txt\x00? untracked.txt\x00")
	entries, err := parsePorcelainV2(input)
	if err != nil {
		t.Fatal(err)
	}
	want := []GitEntry{
		{Kind: "ordinary", XY: ".M", Submodule: "N...", Path: "file.txt"},
		{Kind: "rename", XY: "R.", Submodule: "N...", Path: "new.txt", OriginalPath: "old.txt"},
		{Kind: "untracked", Path: "untracked.txt"},
	}
	if !reflect.DeepEqual(entries, want) {
		t.Fatalf("entries=%+v", entries)
	}
}

func TestCollectGitFixture(t *testing.T) {
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git unavailable")
	}
	root := t.TempDir()
	run := func(args ...string) {
		command := exec.Command(git, append([]string{"-C", root}, args...)...)
		command.Env = append(os.Environ(), "GIT_AUTHOR_NAME=AFR", "GIT_AUTHOR_EMAIL=afr@example.invalid", "GIT_COMMITTER_NAME=AFR", "GIT_COMMITTER_EMAIL=afr@example.invalid")
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
	run("init", "-q")
	if err := os.WriteFile(filepath.Join(root, "tracked.txt"), []byte("base"), 0o600); err != nil {
		t.Fatal(err)
	}
	run("add", "tracked.txt")
	run("commit", "-qm", "base")
	if err := os.WriteFile(filepath.Join(root, "tracked.txt"), []byte("dirty"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "new.txt"), []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	evidence := collectGit(root)
	if !evidence.Available || evidence.Head == "" || len(evidence.Entries) != 2 {
		t.Fatalf("evidence=%+v", evidence)
	}
}

func TestCollectGitStatusMatrix(t *testing.T) {
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git unavailable")
	}
	repo := t.TempDir()
	subrepo := t.TempDir()
	runAt := func(root string, args ...string) {
		command := exec.Command(git, append([]string{"-C", root}, args...)...)
		command.Env = append(os.Environ(), "GIT_AUTHOR_NAME=AFR", "GIT_AUTHOR_EMAIL=afr@example.invalid", "GIT_COMMITTER_NAME=AFR", "GIT_COMMITTER_EMAIL=afr@example.invalid")
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
	runAt(subrepo, "init", "-q")
	if err := os.WriteFile(filepath.Join(subrepo, "sub.txt"), []byte("base"), 0o600); err != nil {
		t.Fatal(err)
	}
	runAt(subrepo, "add", ".")
	runAt(subrepo, "commit", "-qm", "base")
	runAt(repo, "init", "-q")
	for _, name := range []string{"staged.txt", "unstaged.txt", "rename.txt", "delete.txt", "binary.bin"} {
		if err := os.WriteFile(filepath.Join(repo, name), []byte("base"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	runAt(repo, "-c", "protocol.file.allow=always", "submodule", "add", "-q", subrepo, "sub")
	runAt(repo, "add", ".")
	runAt(repo, "commit", "-qm", "base")
	if clean := collectGit(repo); !clean.Available || clean.Error != "" || len(clean.Entries) != 0 {
		t.Fatalf("clean evidence=%+v", clean)
	}
	if err := os.WriteFile(filepath.Join(repo, "staged.txt"), []byte("staged"), 0o600); err != nil {
		t.Fatal(err)
	}
	runAt(repo, "add", "staged.txt")
	if err := os.WriteFile(filepath.Join(repo, "unstaged.txt"), []byte("unstaged"), 0o600); err != nil {
		t.Fatal(err)
	}
	runAt(repo, "mv", "rename.txt", "renamed.txt")
	if err := os.Remove(filepath.Join(repo, "delete.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "binary.bin"), []byte{0, 1, 2, 3}, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "untracked.txt"), []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "sub", "sub.txt"), []byte("dirty submodule"), 0o600); err != nil {
		t.Fatal(err)
	}
	evidence := collectGit(repo)
	paths := map[string]GitEntry{}
	for _, entry := range evidence.Entries {
		paths[entry.Path] = entry
	}
	for _, path := range []string{"staged.txt", "unstaged.txt", "renamed.txt", "delete.txt", "binary.bin", "untracked.txt", "sub"} {
		if _, exists := paths[path]; !exists {
			t.Fatalf("missing %s in %+v", path, evidence.Entries)
		}
	}
	if paths["staged.txt"].XY[0] == '.' || paths["unstaged.txt"].XY[1] == '.' || paths["renamed.txt"].OriginalPath != "rename.txt" || paths["sub"].Submodule == "N..." {
		t.Fatalf("status details=%+v", paths)
	}
}
