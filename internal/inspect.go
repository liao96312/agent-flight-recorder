package afr

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

type EventInspection struct {
	ValidEvents   uint64 `json:"valid_events"`
	FinalHash     string `json:"final_hash"`
	LastEventType string `json:"last_event_type"`
	TornTailBytes int    `json:"torn_tail_bytes"`
	TornTailHash  string `json:"torn_tail_sha256,omitempty"`
}

type SessionSummary struct {
	ID            string `json:"id"`
	State         string `json:"state"`
	StartedAt     string `json:"started_at,omitempty"`
	ChildExitCode *int   `json:"child_exit_code,omitempty"`
	Incomplete    bool   `json:"incomplete"`
	TornTailBytes int    `json:"torn_tail_bytes,omitempty"`
}

const maxEventLineBytes = 8 * 1024 * 1024

func InspectEvents(sessionRoot string) (EventInspection, error) {
	inspection, issue, err := inspectEventStream(sessionRoot)
	if err != nil {
		return inspection, err
	}
	if issue != nil && issue.Code != "event_torn_tail" {
		return inspection, errors.New(issue.Message)
	}
	return inspection, nil
}

func inspectEventStream(sessionRoot string) (EventInspection, *VerificationIssue, error) {
	path, err := safeJoin(sessionRoot, "events.jsonl")
	if err != nil {
		return EventInspection{}, nil, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return EventInspection{}, nil, fmt.Errorf("inspect events: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return EventInspection{}, nil, errors.New("events must be a regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return EventInspection{}, nil, fmt.Errorf("open events: %w", err)
	}
	defer file.Close()

	reader := bufio.NewReaderSize(file, 64*1024)
	inspection := EventInspection{}
	var previous [sha256.Size]byte
	for {
		line, readErr, tooLarge := readBoundedEventLine(reader)
		if tooLarge {
			return inspection, &VerificationIssue{Kind: "event", Code: "event_too_large", Seq: inspection.ValidEvents + 1, Path: "events.jsonl", Message: "event exceeds the verification line limit", Expected: fmt.Sprintf("at most %d bytes", maxEventLineBytes), Actual: "larger event"}, nil
		}
		if readErr == io.EOF && len(line) > 0 {
			sum := sha256.Sum256(line)
			inspection.TornTailBytes = len(line)
			inspection.TornTailHash = hex.EncodeToString(sum[:])
			return inspection, &VerificationIssue{Kind: "event", Code: "event_torn_tail", Seq: inspection.ValidEvents + 1, Path: "events.jsonl", Message: "events file has a torn final record", Expected: "newline-terminated event", Actual: fmt.Sprintf("%d trailing bytes", len(line))}, nil
		}
		if len(line) > 0 {
			raw := bytes.TrimSuffix(line, []byte{'\n'})
			var envelope eventEnvelope
			if err := json.Unmarshal(raw, &envelope); err != nil {
				return inspection, &VerificationIssue{Kind: "event", Code: "event_invalid_json", Seq: inspection.ValidEvents + 1, Path: "events.jsonl", Message: fmt.Sprintf("event %d is invalid JSON", inspection.ValidEvents+1)}, nil
			}
			if envelope.FormatVersion != 1 {
				return inspection, &VerificationIssue{Kind: "event", Code: "event_version", Seq: inspection.ValidEvents + 1, Path: "events.jsonl", Message: fmt.Sprintf("event %d has unsupported format", inspection.ValidEvents+1), Expected: "1", Actual: fmt.Sprint(envelope.FormatVersion)}, nil
			}
			var body EventBody
			if err := json.Unmarshal(envelope.Body, &body); err != nil {
				return inspection, &VerificationIssue{Kind: "event", Code: "event_body_invalid", Seq: inspection.ValidEvents + 1, Path: "events.jsonl", Message: fmt.Sprintf("event %d body is invalid", inspection.ValidEvents+1)}, nil
			}
			if body.Seq != inspection.ValidEvents+1 {
				return inspection, &VerificationIssue{Kind: "event", Code: "event_sequence", Seq: inspection.ValidEvents + 1, Path: "events.jsonl", Message: fmt.Sprintf("event %d sequence is invalid", inspection.ValidEvents+1), Expected: fmt.Sprint(inspection.ValidEvents + 1), Actual: fmt.Sprint(body.Seq)}, nil
			}
			expectedPrevious := hex.EncodeToString(previous[:])
			if envelope.PrevHash != expectedPrevious {
				return inspection, &VerificationIssue{Kind: "event", Code: "event_previous_hash", Seq: body.Seq, Path: "events.jsonl", Message: fmt.Sprintf("event %d previous hash mismatch", body.Seq), Expected: expectedPrevious, Actual: envelope.PrevHash}, nil
			}
			h := sha256.New()
			_, _ = h.Write([]byte("AFR-EVENT-v1\n"))
			_, _ = h.Write(previous[:])
			_, _ = h.Write([]byte("\n"))
			_, _ = h.Write(envelope.Body)
			actual := h.Sum(nil)
			if envelope.Hash != hex.EncodeToString(actual) {
				return inspection, &VerificationIssue{Kind: "event", Code: "event_hash", Seq: body.Seq, Path: "events.jsonl", Message: fmt.Sprintf("event %d hash mismatch", body.Seq), Expected: hex.EncodeToString(actual), Actual: envelope.Hash}, nil
			}
			copy(previous[:], actual)
			inspection.ValidEvents++
			inspection.FinalHash = envelope.Hash
			inspection.LastEventType = body.Type
		}
		if readErr != nil {
			if readErr == io.EOF {
				return inspection, nil, nil
			}
			return inspection, nil, fmt.Errorf("read events: %w", readErr)
		}
	}
}

func readBoundedEventLine(reader *bufio.Reader) ([]byte, error, bool) {
	line := []byte{}
	for {
		fragment, err := reader.ReadSlice('\n')
		if len(line)+len(fragment) > maxEventLineBytes {
			return nil, nil, true
		}
		line = append(line, fragment...)
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		return line, err, false
	}
}

func ListSessions(sessionsRoot string) ([]SessionSummary, error) {
	entries, err := os.ReadDir(sessionsRoot)
	if errors.Is(err, os.ErrNotExist) {
		return []SessionSummary{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read sessions: %w", err)
	}
	summaries := make([]SessionSummary, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || !filepath.IsLocal(entry.Name()) {
			continue
		}
		root, err := safeJoin(sessionsRoot, entry.Name())
		if err != nil {
			continue
		}
		summary := SessionSummary{ID: entry.Name(), State: "incomplete", Incomplete: true}
		metadataPath, err := safeJoin(root, "session.json")
		if err == nil {
			if data, readErr := os.ReadFile(metadataPath); readErr == nil {
				var metadata SessionMetadata
				if json.Unmarshal(data, &metadata) == nil {
					summary.ID = metadata.ID
					summary.State = metadata.State
					summary.StartedAt = metadata.StartedAt
					summary.ChildExitCode = metadata.ChildExitCode
				}
			}
		}
		inspection, inspectErr := InspectEvents(root)
		if inspectErr != nil {
			summary.State = "incomplete"
		} else {
			summary.TornTailBytes = inspection.TornTailBytes
			summary.Incomplete = inspection.TornTailBytes > 0 || inspection.LastEventType != "session_finished" || (summary.State != "completed" && summary.State != "failed")
			if summary.Incomplete {
				summary.State = "incomplete"
			}
		}
		summaries = append(summaries, summary)
	}
	sort.Slice(summaries, func(i, j int) bool { return summaries[i].ID > summaries[j].ID })
	return summaries, nil
}
