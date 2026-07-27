package afr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"testing"
)

func TestCodexHookTwoTurnsReuseSessionAndVerify(t *testing.T) {
	workspace, sessions, pluginData := t.TempDir(), t.TempDir(), t.TempDir()
	hostSession := "019f-hook-session"
	call := func(event map[string]any) HookResult {
		event["session_id"] = hostSession
		event["cwd"] = workspace
		data, err := json.Marshal(event)
		if err != nil {
			t.Fatal(err)
		}
		result, err := IngestCodexHook(HookOptions{SessionsRoot: sessions, PluginData: pluginData, Input: bytes.NewReader(data)})
		if err != nil {
			t.Fatalf("ingest %v: %v", event["hook_event_name"], err)
		}
		return result
	}

	started := call(map[string]any{"hook_event_name": "SessionStart", "source": "resume", "model": "gpt-test"})
	call(map[string]any{"hook_event_name": "UserPromptSubmit", "turn_id": "turn-1", "prompt": "api_key=supersecretvalue"})
	call(map[string]any{"hook_event_name": "PreToolUse", "turn_id": "turn-1", "tool_name": "Bash", "tool_use_id": "tool-1", "tool_input": map[string]any{"command": "echo ok"}, "future_object": map[string]any{"private": "ignored"}})
	call(map[string]any{"hook_event_name": "PostToolUse", "turn_id": "turn-1", "tool_name": "Bash", "tool_use_id": "tool-1", "tool_response": map[string]any{"exit_code": 0}})
	firstStop := call(map[string]any{"hook_event_name": "Stop", "turn_id": "turn-1", "last_assistant_message": "done"})
	if firstStop.SessionID != started.SessionID || firstStop.State != "idle" {
		t.Fatalf("first stop = %+v, started = %+v", firstStop, started)
	}
	if verification := VerifySession(firstStop.SessionDir); !verification.Valid {
		t.Fatalf("first verify: %+v", verification.Issue)
	}
	firstMetadata, issue := readSessionForVerify(firstStop.SessionDir)
	if issue != nil || !slices.Contains(firstMetadata.Capabilities, "native_tool_events=observed") {
		t.Fatalf("first capabilities = %v, issue = %+v", firstMetadata.Capabilities, issue)
	}

	call(map[string]any{"hook_event_name": "SessionStart", "source": "compact"})
	secondPrompt := call(map[string]any{"hook_event_name": "UserPromptSubmit", "turn_id": "turn-2", "prompt": "continue"})
	secondStop := call(map[string]any{"hook_event_name": "Stop", "turn_id": "turn-2"})
	if secondPrompt.SessionID != started.SessionID || secondStop.SessionID != started.SessionID {
		t.Fatalf("session changed: start=%s prompt=%s stop=%s", started.SessionID, secondPrompt.SessionID, secondStop.SessionID)
	}
	if verification := VerifySession(secondStop.SessionDir); !verification.Valid {
		t.Fatalf("second verify: %+v", verification.Issue)
	}

	events, err := os.ReadFile(filepath.Join(secondStop.SessionDir, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{hostSession, "supersecretvalue", `"private":"ignored"`} {
		if bytes.Contains(events, []byte(forbidden)) {
			t.Fatalf("events persisted forbidden value %q", forbidden)
		}
	}
	if !bytes.Contains(events, []byte(`"reason":"unknown_field"`)) {
		t.Fatal("unknown field omission was not recorded")
	}
	mappings, err := filepath.Glob(filepath.Join(pluginData, "sessions", "*.json"))
	if err != nil || len(mappings) != 1 {
		t.Fatalf("mappings = %v, %v", mappings, err)
	}
	mappingData, _ := os.ReadFile(mappings[0])
	if bytes.Contains(mappingData, []byte(hostSession)) {
		t.Fatal("mapping persisted raw host session")
	}

	cleared := call(map[string]any{"hook_event_name": "SessionStart", "source": "clear"})
	if cleared.SessionID == started.SessionID {
		t.Fatal("clear reused the previous AFR session")
	}
	clearedStop := call(map[string]any{"hook_event_name": "Stop", "turn_id": "turn-3"})
	clearedMetadata, issue := readSessionForVerify(clearedStop.SessionDir)
	if issue != nil || !slices.Contains(clearedMetadata.Capabilities, "native_tool_events=not_observable") {
		t.Fatalf("cleared capabilities = %v, issue = %+v", clearedMetadata.Capabilities, issue)
	}
}

func TestCodexHookConcurrentAppend(t *testing.T) {
	workspace, sessions, pluginData := t.TempDir(), t.TempDir(), t.TempDir()
	call := func(event map[string]any) (HookResult, error) {
		event["session_id"] = "concurrent-session"
		event["cwd"] = workspace
		data, _ := json.Marshal(event)
		return IngestCodexHook(HookOptions{SessionsRoot: sessions, PluginData: pluginData, Input: bytes.NewReader(data)})
	}
	started, err := call(map[string]any{"hook_event_name": "SessionStart", "source": "startup"})
	if err != nil {
		t.Fatal(err)
	}
	const workers = 20
	var wait sync.WaitGroup
	errorsFound := make(chan error, workers)
	for index := range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, err := call(map[string]any{"hook_event_name": "PreToolUse", "turn_id": fmt.Sprintf("turn-%d", index), "tool_name": "Bash", "tool_use_id": fmt.Sprintf("tool-%d", index)})
			errorsFound <- err
		}()
	}
	wait.Wait()
	close(errorsFound)
	for err := range errorsFound {
		if err != nil {
			t.Fatal(err)
		}
	}
	stopped, err := call(map[string]any{"hook_event_name": "Stop", "turn_id": "final-turn"})
	if err != nil {
		t.Fatal(err)
	}
	if stopped.SessionID != started.SessionID {
		t.Fatalf("session changed: %s -> %s", started.SessionID, stopped.SessionID)
	}
	verification := VerifySession(stopped.SessionDir)
	if !verification.Valid || verification.Events != workers+4 {
		t.Fatalf("verify = %+v", verification)
	}
}

func TestHookLockRecognizesWindowsContention(t *testing.T) {
	if !hookLockContended(&os.PathError{Err: os.ErrExist}) {
		t.Fatal("existing lock was not treated as contention")
	}
	got := hookLockContended(&os.PathError{Err: os.ErrPermission})
	if got != (runtime.GOOS == "windows") {
		t.Fatalf("permission contention = %t on %s", got, runtime.GOOS)
	}
}

func TestCodexHookReportKeepsPriorTurnRisk(t *testing.T) {
	workspace, sessions, pluginData := t.TempDir(), t.TempDir(), t.TempDir()
	call := func(event map[string]any) HookResult {
		event["session_id"] = "risk-history-session"
		event["cwd"] = workspace
		data, _ := json.Marshal(event)
		result, err := IngestCodexHook(HookOptions{SessionsRoot: sessions, PluginData: pluginData, Input: bytes.NewReader(data)})
		if err != nil {
			t.Fatal(err)
		}
		return result
	}

	call(map[string]any{"hook_event_name": "SessionStart", "source": "startup"})
	for index := range bulkChangeThreshold {
		path := filepath.Join(workspace, fmt.Sprintf("file-%03d.txt", index))
		if err := os.WriteFile(path, []byte("changed"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	call(map[string]any{"hook_event_name": "Stop", "turn_id": "turn-1"})
	for index := range bulkChangeThreshold {
		if err := os.Remove(filepath.Join(workspace, fmt.Sprintf("file-%03d.txt", index))); err != nil {
			t.Fatal(err)
		}
	}
	call(map[string]any{"hook_event_name": "UserPromptSubmit", "turn_id": "turn-2", "prompt": "continue"})
	second := call(map[string]any{"hook_event_name": "Stop", "turn_id": "turn-2"})

	data, err := os.ReadFile(filepath.Join(second.SessionDir, "agent-risk.json"))
	if err != nil {
		t.Fatal(err)
	}
	var report RiskReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.HighestRisk != "medium" || len(report.Findings) != 1 || report.Findings[0].RuleID != "workspace.bulk_change" {
		t.Fatalf("risk report after second turn = %+v", report)
	}
}
