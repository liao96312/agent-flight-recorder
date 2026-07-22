package afr

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestAFRIgnoreFiltersPrefixesAndStandardLibraryGlobs(t *testing.T) {
	root := t.TempDir()
	writeFixture := func(name string) {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"keep.txt", "root.log", "nested/keep.log", "vendor/file.txt", "generated/drop.tmp"} {
		writeFixture(name)
	}
	ignoreFile := "# comment\n\n vendor/\n*.log\ngenerated\\*.tmp\n"
	if err := os.WriteFile(filepath.Join(root, ".afrignore"), []byte(ignoreFile), 0o600); err != nil {
		t.Fatal(err)
	}

	fingerprinter, _ := NewFingerprinter()
	snapshot := CollectWorkspace(root, "", fingerprinter, ScanLimits{})
	paths := make([]string, 0, len(snapshot.Files))
	for _, file := range snapshot.Files {
		paths = append(paths, file.Path)
	}
	want := []string{".afrignore", "keep.txt", "nested/keep.log"}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("paths=%v want=%v", paths, want)
	}
}

func TestAFRIgnoreRejectsInvalidGlob(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".afrignore"), []byte("[invalid\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(RunOptions{Workspace: root}, []string{"child-must-not-start"}); err == nil || !strings.Contains(err.Error(), "invalid .afrignore pattern") {
		t.Fatalf("run error=%v", err)
	}
}
