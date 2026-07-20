package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRunProducesConfiguredFixture(t *testing.T) {
	var stdout, stderr bytes.Buffer
	path := filepath.Join(t.TempDir(), "written.txt")
	code := run([]string{
		"--stdout", "out",
		"--stderr", "err",
		"--long-line", "8",
		"--binary",
		"--write-file", path,
		"--write-text", "file",
		"--exit", "7",
	}, &stdout, &stderr)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if code != 7 || stdout.String() != "outxxxxxxxx\x00\x01\x02\xff" || stderr.String() != "err" || string(data) != "file" {
		t.Fatalf("code=%d stdout=%q stderr=%q file=%q", code, stdout.String(), stderr.String(), data)
	}
}
