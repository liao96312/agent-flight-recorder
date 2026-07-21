package afr

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestWriteManifestSeparatesAndHashesArtifacts(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "snapshots"), 0o700); err != nil {
		t.Fatal(err)
	}
	artifacts := map[string]string{
		"events.jsonl":          "event\n",
		"session.json":          "session\n",
		"snapshots/before.json": "before\n",
		"report.html":           "derived\n",
	}
	for name, content := range artifacts {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	redactor := testRedactor(t)
	if err := WriteManifest(root, "session-id", []string{"session.json", "events.jsonl", "snapshots/before.json"}, []string{"report.html"}, redactor); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	wantPaths := []string{"events.jsonl", "session.json", "snapshots/before.json"}
	gotPaths := []string{manifest.Evidence[0].Path, manifest.Evidence[1].Path, manifest.Evidence[2].Path}
	if manifest.FormatVersion != 1 || !reflect.DeepEqual(gotPaths, wantPaths) || len(manifest.Derived) != 1 {
		t.Fatalf("manifest=%+v", manifest)
	}
	wantHash := sha256.Sum256([]byte(artifacts[wantPaths[0]]))
	if manifest.Evidence[0].SHA256 != hex.EncodeToString(wantHash[:]) || manifest.Evidence[0].Size != int64(len(artifacts[wantPaths[0]])) {
		t.Fatalf("entry=%+v", manifest.Evidence[0])
	}
	for _, entry := range append(manifest.Evidence, manifest.Derived...) {
		if entry.Path == "manifest.json" {
			t.Fatal("manifest included itself")
		}
	}
}

func TestManifestRejectsSelfReference(t *testing.T) {
	if err := WriteManifest(t.TempDir(), "id", []string{"manifest.json"}, nil, testRedactor(t)); err == nil {
		t.Fatal("self reference accepted")
	}
}
