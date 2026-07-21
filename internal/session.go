package afr

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type SessionMetadata struct {
	FormatVersion int      `json:"format_version"`
	ID            string   `json:"id"`
	State         string   `json:"state"`
	StartedAt     string   `json:"started_at"`
	FinishedAt    string   `json:"finished_at,omitempty"`
	Workspace     string   `json:"workspace"`
	Executable    string   `json:"executable"`
	ArgCount      int      `json:"arg_count"`
	RecorderPID   int      `json:"recorder_pid,omitempty"`
	ChildExitCode *int     `json:"child_exit_code,omitempty"`
	FinalEventSeq uint64   `json:"final_event_seq,omitempty"`
	FinalHash     string   `json:"final_hash,omitempty"`
	Capabilities  []string `json:"capabilities"`
	FlushPolicy   struct {
		Bytes        int  `json:"bytes"`
		IntervalMS   int  `json:"interval_ms"`
		CriticalSync bool `json:"critical_sync"`
	} `json:"event_flush"`
}

type Session struct {
	Root     string
	Meta     SessionMetadata
	redactor *Redactor
}

func DefaultSessionsRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find user home: %w", err)
	}
	return filepath.Join(home, ".afr", "sessions"), nil
}

func NewSession(sessionsRoot, workspace string, argv []string, redactor *Redactor) (*Session, error) {
	if len(argv) == 0 {
		return nil, errors.New("empty command")
	}
	if redactor == nil {
		return nil, errors.New("session requires a redactor")
	}
	if err := os.MkdirAll(sessionsRoot, 0o700); err != nil {
		return nil, fmt.Errorf("create sessions root: %w", err)
	}
	canonicalRoot, err := filepath.EvalSymlinks(sessionsRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve sessions root: %w", err)
	}

	var id, root string
	for range 10 {
		id, err = newSessionID(time.Now().UTC())
		if err != nil {
			return nil, err
		}
		root, err = safeJoin(canonicalRoot, id)
		if err != nil {
			return nil, err
		}
		err = os.Mkdir(root, 0o700)
		if err == nil {
			break
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("create session directory: %w", err)
		}
	}
	if err != nil {
		return nil, errors.New("could not allocate unique session id")
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	s := &Session{Root: root, redactor: redactor, Meta: SessionMetadata{
		FormatVersion: 1,
		ID:            id,
		State:         "starting",
		StartedAt:     now,
		Workspace:     workspace,
		Executable:    filepath.Base(argv[0]),
		ArgCount:      len(argv) - 1,
		RecorderPID:   os.Getpid(),
		Capabilities: []string{
			"process=observed",
			"workspace_git=not_observable",
			"workspace_scan=not_observable",
			"native_tool_events=not_observable",
			"local_policy=observed",
			"os_file_monitor=not_observable",
			"network_monitor=not_observable",
		},
	}}
	s.Meta.FlushPolicy.Bytes = flushBytes
	s.Meta.FlushPolicy.IntervalMS = int(flushInterval / time.Millisecond)
	s.Meta.FlushPolicy.CriticalSync = true
	if err := atomicWriteJSON(root, "session.json", s.Meta, redactor); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Session) Finish(state string, exitCode *int, seq uint64, finalHash string) error {
	s.Meta.State = state
	s.Meta.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	s.Meta.ChildExitCode = exitCode
	s.Meta.RecorderPID = 0
	s.Meta.FinalEventSeq = seq
	s.Meta.FinalHash = finalHash
	return atomicWriteJSON(s.Root, "session.json", s.Meta, s.redactor)
}

func newSessionID(now time.Time) (string, error) {
	random := make([]byte, 12)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}
	return now.UTC().Format("20060102T150405.000000000Z") + "-" + hex.EncodeToString(random), nil
}

func safeJoin(root, relative string) (string, error) {
	if relative == "" || !filepath.IsLocal(relative) {
		return "", errors.New("path must be a non-empty relative path")
	}
	for _, part := range strings.Split(strings.ReplaceAll(relative, "\\", "/"), "/") {
		if part == ".." {
			return "", errors.New("path escapes session root")
		}
	}
	clean := filepath.Clean(relative)
	if clean == "." {
		return "", errors.New("path escapes session root")
	}
	target := filepath.Join(root, clean)
	rel, err := filepath.Rel(root, target)
	if err != nil || !filepath.IsLocal(rel) {
		return "", errors.New("path escapes session root")
	}
	return target, nil
}

func atomicWriteJSON(root, relative string, value any, redactor *Redactor) error {
	if redactor == nil {
		return errors.New("persistent JSON requires a redactor")
	}
	data, err := redactor.MarshalIndent(value)
	if err != nil {
		return fmt.Errorf("encode %s: %w", relative, err)
	}
	data = append(data, '\n')
	return atomicWriteBytes(root, relative, data)
}

func atomicWriteRedactedText(root, relative, text string, redactor *Redactor) error {
	if redactor == nil {
		return errors.New("persistent text requires a redactor")
	}
	return atomicWriteBytes(root, relative, []byte(redactor.Text(text)))
}

func atomicWriteBytes(root, relative string, data []byte) error {
	target, err := safeJoin(root, relative)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(target), ".afr-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary %s: %w", relative, err)
	}
	tmpName := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}()
	if err := tmp.Chmod(0o600); err != nil {
		return fmt.Errorf("restrict temporary %s: %w", relative, err)
	}
	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("write temporary %s: %w", relative, err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync temporary %s: %w", relative, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary %s: %w", relative, err)
	}
	if err := os.Rename(tmpName, target); err != nil {
		return fmt.Errorf("replace %s: %w", relative, err)
	}
	return nil
}
