package afr

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifySessionAcceptsUntamperedEvidence(t *testing.T) {
	root := createVerifiableSession(t, true)
	result := VerifySession(root)
	if !result.Valid || result.Issue != nil || result.Events != 2 || result.EvidenceChecked != len(requiredEvidencePaths) || result.DerivedChecked != 1 {
		t.Fatalf("result=%+v", result)
	}
}

func TestVerifyLocatesFirstEventAnomaly(t *testing.T) {
	tests := []struct {
		name string
		want uint64
		code string
		edit func([][]byte) []byte
	}{
		{name: "delete", want: 1, code: "event_sequence", edit: func(lines [][]byte) []byte { return joinEventLines(lines[1:]) }},
		{name: "delete_tail", want: 2, code: "event_count", edit: func(lines [][]byte) []byte { return joinEventLines(lines[:1]) }},
		{name: "insert", want: 2, code: "event_sequence", edit: func(lines [][]byte) []byte { return joinEventLines([][]byte{lines[0], lines[0], lines[1]}) }},
		{name: "rewrite", want: 1, code: "event_hash", edit: func(lines [][]byte) []byte {
			lines[0] = bytes.Replace(lines[0], []byte("session_started"), []byte("session_starteD"), 1)
			return joinEventLines(lines)
		}},
		{name: "reorder", want: 1, code: "event_sequence", edit: func(lines [][]byte) []byte { return joinEventLines([][]byte{lines[1], lines[0]}) }},
		{name: "torn", want: 3, code: "event_torn_tail", edit: func(lines [][]byte) []byte { return append(joinEventLines(lines), '{') }},
		{name: "oversized", want: 1, code: "event_too_large", edit: func(lines [][]byte) []byte { return bytes.Repeat([]byte{'x'}, maxEventLineBytes+1) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := createVerifiableSession(t, false)
			path := filepath.Join(root, "events.jsonl")
			data, _ := os.ReadFile(path)
			lines := bytes.Split(bytes.TrimSuffix(data, []byte{'\n'}), []byte{'\n'})
			if err := os.WriteFile(path, test.edit(lines), 0o600); err != nil {
				t.Fatal(err)
			}
			result := VerifySession(root)
			if result.Valid || result.Issue == nil || result.Issue.Seq != test.want || result.Issue.Code != test.code {
				t.Fatalf("result=%+v", result)
			}
		})
	}
}

func TestVerifyLocatesArtifactAnomaly(t *testing.T) {
	artifacts := []string{"diffs/workspace.patch", "snapshots/workspace-after.json", "report.html"}
	mutations := []struct {
		name string
		code string
		edit func(string) error
	}{
		{name: "missing", code: "artifact_missing", edit: os.Remove},
		{name: "replaced", code: "artifact_mismatch", edit: func(path string) error { return os.WriteFile(path, []byte("replacement\n"), 0o600) }},
		{name: "truncated", code: "artifact_mismatch", edit: func(path string) error { return os.Truncate(path, 1) }},
	}
	for _, artifact := range artifacts {
		for _, mutation := range mutations {
			t.Run(filepath.Base(artifact)+"_"+mutation.name, func(t *testing.T) {
				root := createVerifiableSession(t, true)
				if err := mutation.edit(filepath.Join(root, filepath.FromSlash(artifact))); err != nil {
					t.Fatal(err)
				}
				result := VerifySession(root)
				if result.Valid || result.Issue == nil || result.Issue.Path != artifact || result.Issue.Code != mutation.code || result.Issue.Want == nil || result.Issue.Got == nil {
					t.Fatalf("result=%+v", result)
				}
			})
		}
	}
}

func TestVerifyChecksSessionFinalHash(t *testing.T) {
	root := createVerifiableSession(t, false)
	path := filepath.Join(root, "session.json")
	data, _ := os.ReadFile(path)
	var metadata SessionMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatal(err)
	}
	metadata.FinalHash = "00"
	data, _ = json.Marshal(metadata)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	result := VerifySession(root)
	if result.Valid || result.Issue == nil || result.Issue.Code != "final_hash_mismatch" {
		t.Fatalf("result=%+v", result)
	}
}

func TestResolveSessionRootLatest(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"20260101-a", "20260102-b"} {
		if err := os.Mkdir(filepath.Join(root, name), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	latest, err := ResolveSessionRoot(root, "latest")
	if err != nil || filepath.Base(latest) != "20260102-b" {
		t.Fatalf("latest=%q error=%v", latest, err)
	}
	if _, err := ResolveSessionRoot(root, "../escape"); err == nil {
		t.Fatal("escaping selector accepted")
	}
}

func createVerifiableSession(t *testing.T, withDerived bool) string {
	t.Helper()
	redactor := testRedactor(t)
	session, err := NewSession(t.TempDir(), t.TempDir(), []string{"agent"}, redactor)
	if err != nil {
		t.Fatal(err)
	}
	writer, err := NewEventWriter(session.Root, redactor)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Append("session_started", "afr", map[string]string{"id": session.Meta.ID}, true); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Append("session_finished", "afr", map[string]string{"state": "completed"}, true); err != nil {
		t.Fatal(err)
	}
	seq, finalHash := writer.Snapshot()
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	exitCode := 0
	if err := session.Finish("completed", &exitCode, seq, finalHash); err != nil {
		t.Fatal(err)
	}
	for _, directory := range []string{"diffs", "snapshots"} {
		if err := os.Mkdir(filepath.Join(session.Root, directory), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	for _, relative := range requiredEvidencePaths {
		path := filepath.Join(session.Root, filepath.FromSlash(relative))
		if _, err := os.Stat(path); err == nil {
			continue
		}
		if err := os.WriteFile(path, []byte(relative+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	derived := []string{}
	if withDerived {
		if err := os.WriteFile(filepath.Join(session.Root, "report.html"), []byte("report\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		derived = append(derived, "report.html")
	}
	if err := WriteManifest(session.Root, session.Meta.ID, requiredEvidencePaths, derived, redactor); err != nil {
		t.Fatal(err)
	}
	return session.Root
}

func joinEventLines(lines [][]byte) []byte {
	return append(bytes.Join(lines, []byte{'\n'}), '\n')
}
