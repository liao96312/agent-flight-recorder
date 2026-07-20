package afr

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

const (
	flushBytes    = 64 * 1024
	flushInterval = time.Second
)

type EventBody struct {
	Seq       uint64          `json:"seq"`
	Timestamp string          `json:"ts"`
	ElapsedNS int64           `json:"elapsed_ns"`
	Type      string          `json:"type"`
	Source    string          `json:"source"`
	Payload   json.RawMessage `json:"payload"`
}

type eventEnvelope struct {
	FormatVersion int             `json:"format_version"`
	PrevHash      string          `json:"prev_hash"`
	Body          json.RawMessage `json:"body"`
	Hash          string          `json:"hash"`
}

type EventWriter struct {
	mu        sync.Mutex
	file      *os.File
	buffer    *bufio.Writer
	started   time.Time
	seq       uint64
	prevHash  [sha256.Size]byte
	pending   int
	lastFlush time.Time
	closed    bool
}

func NewEventWriter(sessionRoot string) (*EventWriter, error) {
	path, err := safeJoin(sessionRoot, "events.jsonl")
	if err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, fmt.Errorf("create events: %w", err)
	}
	now := time.Now()
	return &EventWriter{file: file, buffer: bufio.NewWriterSize(file, flushBytes), started: now, lastFlush: now}, nil
}

func (w *EventWriter) Append(eventType, source string, payload any, critical bool) (uint64, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return 0, os.ErrClosed
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("encode event payload: %w", err)
	}
	body := EventBody{
		Seq:       w.seq + 1,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		ElapsedNS: time.Since(w.started).Nanoseconds(),
		Type:      eventType,
		Source:    source,
		Payload:   payloadBytes,
	}
	line, hash, err := encodeEvent(w.prevHash, body)
	if err != nil {
		return 0, err
	}
	if _, err := w.buffer.Write(append(line, '\n')); err != nil {
		return 0, fmt.Errorf("append event: %w", err)
	}
	w.seq = body.Seq
	w.prevHash = hash
	w.pending += len(line) + 1
	if critical || w.pending >= flushBytes || time.Since(w.lastFlush) >= flushInterval {
		if err := w.flush(critical); err != nil {
			return 0, err
		}
	}
	return body.Seq, nil
}

func encodeEvent(prevHash [sha256.Size]byte, body EventBody) ([]byte, [sha256.Size]byte, error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, [sha256.Size]byte{}, fmt.Errorf("encode event body: %w", err)
	}
	h := sha256.New()
	_, _ = h.Write([]byte("AFR-EVENT-v1\n"))
	_, _ = h.Write(prevHash[:])
	_, _ = h.Write([]byte("\n"))
	_, _ = h.Write(bodyBytes)
	var hash [sha256.Size]byte
	copy(hash[:], h.Sum(nil))
	envelope := eventEnvelope{
		FormatVersion: 1,
		PrevHash:      hex.EncodeToString(prevHash[:]),
		Body:          bodyBytes,
		Hash:          hex.EncodeToString(hash[:]),
	}
	line, err := json.Marshal(envelope)
	if err != nil {
		return nil, [sha256.Size]byte{}, fmt.Errorf("encode event: %w", err)
	}
	return line, hash, nil
}

func (w *EventWriter) flush(syncFile bool) error {
	if err := w.buffer.Flush(); err != nil {
		return fmt.Errorf("flush events: %w", err)
	}
	if syncFile {
		if err := w.file.Sync(); err != nil {
			return fmt.Errorf("sync events: %w", err)
		}
	}
	w.pending = 0
	w.lastFlush = time.Now()
	return nil
}

func (w *EventWriter) Snapshot() (uint64, string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.seq, hex.EncodeToString(w.prevHash[:])
}

func (w *EventWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return nil
	}
	w.closed = true
	if err := w.flush(true); err != nil {
		_ = w.file.Close()
		return err
	}
	if err := w.file.Close(); err != nil {
		return fmt.Errorf("close events: %w", err)
	}
	return nil
}
