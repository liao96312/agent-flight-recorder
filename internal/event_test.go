package afr

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEventV1GoldenVector(t *testing.T) {
	body := EventBody{
		Seq:       1,
		Timestamp: "2026-07-20T02:31:18.426Z",
		ElapsedNS: 0,
		Type:      "session_started",
		Source:    "afr",
		Payload:   json.RawMessage(`{"session_id":"test"}`),
	}
	line, _, err := encodeEvent([sha256.Size]byte{}, body)
	if err != nil {
		t.Fatal(err)
	}
	const expected = `{"format_version":1,"prev_hash":"0000000000000000000000000000000000000000000000000000000000000000","body":{"seq":1,"ts":"2026-07-20T02:31:18.426Z","elapsed_ns":0,"type":"session_started","source":"afr","payload":{"session_id":"test"}},"hash":"306c8e543724a0ac4cbe21a148e93b03eeb0ad4ab53b5d8101d7ab8d36cb7d60"}`
	if string(line) != expected {
		t.Fatalf("golden vector changed:\n%s", line)
	}
}

func TestEventWriterHundredThousandChain(t *testing.T) {
	root := t.TempDir()
	writer, err := NewEventWriter(root)
	if err != nil {
		t.Fatal(err)
	}
	for index := 1; index <= 100_000; index++ {
		if seq, err := writer.Append("process_output", "process", map[string]int{"bytes": index}, false); err != nil || seq != uint64(index) {
			t.Fatalf("append %d: seq=%d error=%v", index, seq, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(filepath.Join(root, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	var previous [sha256.Size]byte
	for index := 1; ; index++ {
		line, readErr := reader.ReadBytes('\n')
		if len(line) == 0 && readErr == io.EOF {
			if index != 100_001 {
				t.Fatalf("event count = %d", index-1)
			}
			break
		}
		if readErr != nil && readErr != io.EOF {
			t.Fatal(readErr)
		}
		line = bytes.TrimSuffix(line, []byte{'\n'})
		var envelope eventEnvelope
		if err := json.Unmarshal(line, &envelope); err != nil {
			t.Fatalf("decode %d: %v", index, err)
		}
		if envelope.PrevHash != hex.EncodeToString(previous[:]) {
			t.Fatalf("event %d previous hash mismatch", index)
		}
		h := sha256.New()
		_, _ = h.Write([]byte("AFR-EVENT-v1\n"))
		_, _ = h.Write(previous[:])
		_, _ = h.Write([]byte("\n"))
		_, _ = h.Write(envelope.Body)
		actual := h.Sum(nil)
		if envelope.Hash != hex.EncodeToString(actual) {
			t.Fatalf("event %d hash mismatch", index)
		}
		copy(previous[:], actual)
		if readErr == io.EOF {
			break
		}
	}
}

func TestSafeJoinRejectsEscapes(t *testing.T) {
	root := t.TempDir()
	for _, path := range []string{"", ".", "..", "../other", "snapshots/../session.json", `C:\\outside`, `/outside`} {
		if got, err := safeJoin(root, path); err == nil {
			t.Errorf("safeJoin(%q) = %q, want error", path, got)
		}
	}
	inside, err := safeJoin(root, "snapshots/before.json")
	if err != nil || !strings.HasPrefix(inside, root) {
		t.Fatalf("safe path = %q, %v", inside, err)
	}
}
