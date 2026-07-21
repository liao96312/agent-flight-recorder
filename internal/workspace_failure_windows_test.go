package afr

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

func TestWorkspaceUnreadableFileIsVisible(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "locked.txt")
	if err := os.WriteFile(path, []byte("locked"), 0o600); err != nil {
		t.Fatal(err)
	}
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := windows.CreateFile(name, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(handle)

	fingerprinter, _ := NewFingerprinter()
	snapshot := CollectWorkspace(root, "", fingerprinter, ScanLimits{})
	if !snapshot.Partial || len(snapshot.Files) != 1 || snapshot.Files[0].Path != "locked.txt" || snapshot.Files[0].OmittedReason == "" {
		t.Fatalf("snapshot=%+v", snapshot)
	}
}

func TestWorkspaceMissingLinkTargetIsVisible(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "gone")
	link := filepath.Join(root, "link")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("cmd.exe", "/d", "/c", "mklink", "/J", link, target).CombinedOutput(); err != nil {
		t.Fatalf("create junction: %v: %s", err, output)
	}
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}

	fingerprinter, _ := NewFingerprinter()
	snapshot := CollectWorkspace(root, "", fingerprinter, ScanLimits{})
	if !snapshot.Partial || len(snapshot.Files) != 1 || snapshot.Files[0].Path != "link" || snapshot.Files[0].OmittedReason != "link_target_unresolved" {
		t.Fatalf("snapshot=%+v", snapshot)
	}
}
