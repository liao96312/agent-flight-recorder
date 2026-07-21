package afr

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestRunRecordsOutputAndNonZeroExit(t *testing.T) {
	if exitText := os.Getenv("AFR_TEST_EXIT"); exitText != "" {
		fmt.Fprint(os.Stdout, "hello stdout")
		fmt.Fprint(os.Stderr, "hello stderr")
		exitCode, _ := strconv.Atoi(exitText)
		os.Exit(exitCode)
	}
	t.Setenv("AFR_TEST_EXIT", "7")
	var stdout, stderr bytes.Buffer
	result, err := Run(RunOptions{
		SessionsRoot: filepath.Join(t.TempDir(), "sessions"),
		Workspace:    t.TempDir(),
		Stdout:       &stdout,
		Stderr:       &stderr,
	}, []string{os.Args[0], "-test.run=TestRunRecordsOutputAndNonZeroExit"})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 7 || stdout.String() != "hello stdout" || stderr.String() != "hello stderr" {
		t.Fatalf("result=%+v stdout=%q stderr=%q", result, stdout.String(), stderr.String())
	}
	for _, expected := range []string{"Session: " + result.SessionID, "Evidence: observed=", "Changed: added=", "Risks: total=", "Exit: child=7 state=completed", "Report: " + filepath.Join(result.SessionDir, "report.html")} {
		if !bytes.Contains([]byte(result.Summary), []byte(expected)) {
			t.Fatalf("run summary missing %q: %s", expected, result.Summary)
		}
	}
	data, err := os.ReadFile(filepath.Join(result.SessionDir, "session.json"))
	if err != nil {
		t.Fatal(err)
	}
	var session SessionMetadata
	if err := json.Unmarshal(data, &session); err != nil {
		t.Fatal(err)
	}
	if session.State != "completed" || session.ChildExitCode == nil || *session.ChildExitCode != 7 || session.FinalEventSeq < 6 {
		t.Fatalf("unexpected session: %+v", session)
	}
	events, err := os.ReadFile(filepath.Join(result.SessionDir, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(events, []byte("hello stdout")) || !bytes.Contains(events, []byte("hello stderr")) {
		t.Fatal("safe child output was not persisted")
	}
	var previous [sha256.Size]byte
	var eventTypes []string
	for index, line := range bytes.Split(bytes.TrimSpace(events), []byte{'\n'}) {
		var envelope eventEnvelope
		if err := json.Unmarshal(line, &envelope); err != nil {
			t.Fatalf("decode event %d: %v", index+1, err)
		}
		if envelope.PrevHash != hex.EncodeToString(previous[:]) {
			t.Fatalf("event %d prev_hash = %s", index+1, envelope.PrevHash)
		}
		var body EventBody
		if err := json.Unmarshal(envelope.Body, &body); err != nil {
			t.Fatalf("decode body %d: %v", index+1, err)
		}
		if body.Seq != uint64(index+1) {
			t.Fatalf("event seq = %d, want %d", body.Seq, index+1)
		}
		if len(body.Payload) == 0 {
			t.Fatalf("event %d has empty payload", index+1)
		}
		eventTypes = append(eventTypes, body.Type)
		expected, hash, err := encodeEvent(previous, body)
		if err != nil || !bytes.Equal(expected, line) {
			t.Fatalf("event %d hash chain mismatch: %v", index+1, err)
		}
		previous = hash
	}
	if len(eventTypes) < 8 || eventTypes[0] != "session_started" || eventTypes[1] != "workspace_baseline" || eventTypes[2] != "process_started" || eventTypes[len(eventTypes)-3] != "workspace_final" || eventTypes[len(eventTypes)-2] != "process_exited" || eventTypes[len(eventTypes)-1] != "session_finished" {
		t.Fatalf("unexpected lifecycle: %v", eventTypes)
	}
}

func TestRunSecondInterruptForcesTermination(t *testing.T) {
	if os.Getenv("AFR_TEST_CANCEL") == "1" {
		signal.Ignore(os.Interrupt)
		time.Sleep(30 * time.Second)
		os.Exit(0)
	}
	t.Setenv("AFR_TEST_CANCEL", "1")
	interrupts := make(chan os.Signal, 2)
	interrupts <- os.Interrupt
	interrupts <- os.Interrupt
	started := time.Now()
	result, err := Run(RunOptions{
		SessionsRoot: filepath.Join(t.TempDir(), "sessions"),
		Workspace:    t.TempDir(),
		Interrupts:   interrupts,
		GracePeriod:  time.Hour,
	}, []string{os.Args[0], "-test.run=TestRunSecondInterruptForcesTermination"})
	if err != nil || time.Since(started) > 5*time.Second {
		t.Fatalf("result=%+v elapsed=%s error=%v", result, time.Since(started), err)
	}
	events, err := os.ReadFile(filepath.Join(result.SessionDir, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, line := range bytes.Split(bytes.TrimSpace(events), []byte{'\n'}) {
		var envelope eventEnvelope
		if json.Unmarshal(line, &envelope) != nil {
			continue
		}
		var body EventBody
		if json.Unmarshal(envelope.Body, &body) != nil || body.Type != "session_interrupted" {
			continue
		}
		var payload struct {
			Forced bool `json:"forced"`
		}
		if json.Unmarshal(body.Payload, &payload) != nil || !payload.Forced {
			t.Fatalf("interruption did not record forced result: %s", body.Payload)
		}
		found = true
	}
	if !found {
		t.Fatal("session_interrupted event missing")
	}
}

func TestRunFinalizesAfterChildCrash(t *testing.T) {
	if os.Getenv("AFR_TEST_CRASH") == "1" {
		process, _ := os.FindProcess(os.Getpid())
		_ = process.Kill()
		time.Sleep(time.Second)
		os.Exit(1)
	}
	t.Setenv("AFR_TEST_CRASH", "1")
	result, err := Run(RunOptions{
		SessionsRoot: filepath.Join(t.TempDir(), "sessions"),
		Workspace:    t.TempDir(),
	}, []string{os.Args[0], "-test.run=TestRunFinalizesAfterChildCrash"})
	if err != nil || result.ExitCode == 0 {
		t.Fatalf("result=%+v error=%v", result, err)
	}
	summaries, err := ListSessions(filepath.Dir(result.SessionDir))
	if err != nil || len(summaries) != 1 || summaries[0].Incomplete || summaries[0].State != "completed" {
		t.Fatalf("summaries=%+v error=%v", summaries, err)
	}
}

func TestRunExitCodes(t *testing.T) {
	if exitText := os.Getenv("AFR_TEST_EXIT_TABLE"); exitText != "" {
		exitCode, _ := strconv.Atoi(exitText)
		os.Exit(exitCode)
	}
	for _, exitCode := range []int{0, 1, 7, 127} {
		t.Run(strconv.Itoa(exitCode), func(t *testing.T) {
			t.Setenv("AFR_TEST_EXIT_TABLE", strconv.Itoa(exitCode))
			result, err := Run(RunOptions{
				SessionsRoot: filepath.Join(t.TempDir(), "sessions"),
				Workspace:    t.TempDir(),
			}, []string{os.Args[0], "-test.run=TestRunExitCodes"})
			if err != nil || result.ExitCode != exitCode {
				t.Fatalf("result=%+v error=%v", result, err)
			}
		})
	}
}

func TestRunTenMegabyteLine(t *testing.T) {
	if os.Getenv("AFR_TEST_LONG_LINE") == "1" {
		_, _ = io.CopyN(os.Stdout, zeroReader{}, 10*1024*1024)
		os.Exit(0)
	}
	t.Setenv("AFR_TEST_LONG_LINE", "1")
	counter := &countWriter{}
	result, err := Run(RunOptions{
		SessionsRoot: filepath.Join(t.TempDir(), "sessions"),
		Workspace:    t.TempDir(),
		Stdout:       counter,
	}, []string{os.Args[0], "-test.run=TestRunTenMegabyteLine"})
	if err != nil || result.ExitCode != 0 || counter.bytes != 10*1024*1024 {
		t.Fatalf("result=%+v bytes=%d error=%v", result, counter.bytes, err)
	}
}

type zeroReader struct{}

func (zeroReader) Read(buffer []byte) (int, error) {
	for index := range buffer {
		buffer[index] = 'x'
	}
	return len(buffer), nil
}

type countWriter struct{ bytes int }

func (w *countWriter) Write(buffer []byte) (int, error) {
	w.bytes += len(buffer)
	return len(buffer), nil
}

func TestRunPreservesArgvBoundaries(t *testing.T) {
	if os.Getenv("AFR_TEST_ARGV") == "1" {
		expected := []string{"space arg", `"quoted"`, "中文", "--leading"}
		actual := os.Args[len(os.Args)-len(expected):]
		for index := range expected {
			if actual[index] != expected[index] {
				os.Exit(9)
			}
		}
		os.Exit(0)
	}
	t.Setenv("AFR_TEST_ARGV", "1")
	result, err := Run(RunOptions{
		SessionsRoot: filepath.Join(t.TempDir(), "sessions"),
		Workspace:    t.TempDir(),
	}, []string{os.Args[0], "-test.run=TestRunPreservesArgvBoundaries", "--", "space arg", `"quoted"`, "中文", "--leading"})
	if err != nil || result.ExitCode != 0 {
		t.Fatalf("result=%+v error=%v", result, err)
	}
}

func TestRunDrainsBothStreams(t *testing.T) {
	const size = 1024 * 1024
	if os.Getenv("AFR_TEST_DUAL_STREAM") == "1" {
		payload := bytes.Repeat([]byte{'x'}, size)
		var writes sync.WaitGroup
		writes.Add(2)
		go func() { defer writes.Done(); _, _ = os.Stdout.Write(payload) }()
		go func() { defer writes.Done(); _, _ = os.Stderr.Write(payload) }()
		writes.Wait()
		os.Exit(0)
	}
	t.Setenv("AFR_TEST_DUAL_STREAM", "1")
	stdout, stderr := &countWriter{}, &countWriter{}
	result, err := Run(RunOptions{
		SessionsRoot: filepath.Join(t.TempDir(), "sessions"),
		Workspace:    t.TempDir(),
		Stdout:       stdout,
		Stderr:       stderr,
	}, []string{os.Args[0], "-test.run=TestRunDrainsBothStreams"})
	if err != nil || result.ExitCode != 0 || stdout.bytes != size || stderr.bytes != size {
		t.Fatalf("result=%+v stdout=%d stderr=%d error=%v", result, stdout.bytes, stderr.bytes, err)
	}
}
