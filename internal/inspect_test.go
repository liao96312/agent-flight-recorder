package afr

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestInspectEventsPreservesTornTail(t *testing.T) {
	root := t.TempDir()
	writer, err := NewEventWriter(root, testRedactor(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Append("session_started", "afr", map[string]string{"id": "test"}, true); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "events.jsonl")
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("secret")); err != nil {
		t.Fatal(err)
	}
	file.Close()
	before, _ := os.ReadFile(path)
	inspection, err := InspectEvents(root)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(path)
	if inspection.ValidEvents != 1 || inspection.TornTailBytes != 6 || inspection.TornTailHash == "" || !bytes.Equal(before, after) {
		t.Fatalf("inspection=%+v file changed=%v", inspection, !bytes.Equal(before, after))
	}
}

func TestListInfersIncompleteWithoutWriting(t *testing.T) {
	sessions := filepath.Join(t.TempDir(), "sessions")
	redactor := testRedactor(t)
	session, err := NewSession(sessions, t.TempDir(), []string{"agent"}, redactor)
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
	writer.Close()
	before, _ := os.ReadFile(filepath.Join(session.Root, "session.json"))
	summaries, err := ListSessions(sessions)
	if err != nil || len(summaries) != 1 || !summaries[0].Incomplete || summaries[0].State != "incomplete" {
		t.Fatalf("summaries=%+v error=%v", summaries, err)
	}
	after, _ := os.ReadFile(filepath.Join(session.Root, "session.json"))
	if !bytes.Equal(before, after) {
		t.Fatal("incomplete inference mutated session")
	}
}

func TestListMissingRootIsEmptyNotNull(t *testing.T) {
	summaries, err := ListSessions(filepath.Join(t.TempDir(), "missing"))
	if err != nil || summaries == nil || len(summaries) != 0 {
		t.Fatalf("summaries=%v error=%v", summaries, err)
	}
}

func TestShowIncompleteSessionDoesNotWrite(t *testing.T) {
	sessions := filepath.Join(t.TempDir(), "sessions")
	redactor := testRedactor(t)
	session, err := NewSession(sessions, t.TempDir(), []string{"agent"}, redactor)
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
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(session.Root, "agent-flight.md"), []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(filepath.Join(session.Root, "session.json"))
	beforeEntries, _ := os.ReadDir(session.Root)
	result, err := ShowSession(session.Root)
	if err != nil || !result.Session.Incomplete || result.Session.State != "incomplete" || result.MarkdownPath != "" || result.HTMLPath != "" {
		t.Fatalf("result=%+v error=%v", result, err)
	}
	after, _ := os.ReadFile(filepath.Join(session.Root, "session.json"))
	afterEntries, _ := os.ReadDir(session.Root)
	if !bytes.Equal(before, after) || len(beforeEntries) != len(afterEntries) {
		t.Fatal("show mutated incomplete session")
	}
}
