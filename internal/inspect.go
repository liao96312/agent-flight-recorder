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

func InspectEvents(sessionRoot string) (EventInspection, error) {
	path, err := safeJoin(sessionRoot, "events.jsonl")
	if err != nil {
		return EventInspection{}, err
	}
	file, err := os.Open(path)
	if err != nil {
		return EventInspection{}, fmt.Errorf("open events: %w", err)
	}
	defer file.Close()

	reader := bufio.NewReaderSize(file, 64*1024)
	inspection := EventInspection{}
	var previous [sha256.Size]byte
	for {
		line, readErr := reader.ReadBytes('\n')
		if readErr == io.EOF && len(line) > 0 {
			sum := sha256.Sum256(line)
			inspection.TornTailBytes = len(line)
			inspection.TornTailHash = hex.EncodeToString(sum[:])
			return inspection, nil
		}
		if len(line) > 0 {
			raw := bytes.TrimSuffix(line, []byte{'\n'})
			var envelope eventEnvelope
			if err := json.Unmarshal(raw, &envelope); err != nil {
				return inspection, fmt.Errorf("event %d is invalid JSON: %w", inspection.ValidEvents+1, err)
			}
			if envelope.FormatVersion != 1 || envelope.PrevHash != hex.EncodeToString(previous[:]) {
				return inspection, fmt.Errorf("event %d has invalid version or previous hash", inspection.ValidEvents+1)
			}
			h := sha256.New()
			_, _ = h.Write([]byte("AFR-EVENT-v1\n"))
			_, _ = h.Write(previous[:])
			_, _ = h.Write([]byte("\n"))
			_, _ = h.Write(envelope.Body)
			actual := h.Sum(nil)
			if envelope.Hash != hex.EncodeToString(actual) {
				return inspection, fmt.Errorf("event %d hash mismatch", inspection.ValidEvents+1)
			}
			var body EventBody
			if err := json.Unmarshal(envelope.Body, &body); err != nil || body.Seq != inspection.ValidEvents+1 {
				return inspection, fmt.Errorf("event %d has invalid body or sequence", inspection.ValidEvents+1)
			}
			copy(previous[:], actual)
			inspection.ValidEvents++
			inspection.FinalHash = envelope.Hash
			inspection.LastEventType = body.Type
		}
		if readErr != nil {
			if readErr == io.EOF {
				return inspection, nil
			}
			return inspection, fmt.Errorf("read events: %w", readErr)
		}
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
