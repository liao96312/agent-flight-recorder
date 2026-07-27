package afr

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"
)

const (
	maxHookInputBytes    = 1024 * 1024
	maxHookArtifactBytes = 64 * 1024 * 1024
	hookLockStale        = 30 * time.Second
)

var hostIDPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,256}$`)

type HookOptions struct {
	SessionsRoot string
	PluginData   string
	Input        io.Reader
	ScanLimits   ScanLimits
	LockTimeout  time.Duration
}

type HookResult struct {
	SessionID  string
	SessionDir string
	State      string
}

type codexHook struct {
	EventName      string
	SessionID      string
	TurnID         string
	Source         string
	CWD            string
	ToolName       string
	ToolUseID      string
	PermissionMode string
	Model          string
	Values         map[string]any
	Omitted        []hookOmittedField
}

type hookOmittedField struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Bytes  int    `json:"bytes"`
	SHA256 string `json:"sha256"`
	Reason string `json:"reason"`
}

type hookMapping struct {
	FormatVersion   int    `json:"format_version"`
	HostSessionHash string `json:"host_session_hash"`
	AFRSessionID    string `json:"afr_session_id"`
	Workspace       string `json:"workspace"`
	Key             string `json:"fingerprinter_key"`
}

type hookLockOwner struct {
	PID       int    `json:"pid"`
	CreatedAt string `json:"created_at"`
}

func IngestCodexHook(options HookOptions) (HookResult, error) {
	input, err := parseCodexHook(options.Input)
	if err != nil {
		return HookResult{}, err
	}
	if options.LockTimeout <= 0 {
		options.LockTimeout = 5 * time.Second
	}
	if options.SessionsRoot == "" {
		options.SessionsRoot, err = DefaultSessionsRoot()
		if err != nil {
			return HookResult{}, err
		}
	}
	pluginData, err := resolveHookRoot(options.PluginData)
	if err != nil {
		return HookResult{}, err
	}
	hostHash := hashText(input.SessionID)
	var result HookResult
	err = withHookLock(pluginData, hostHash, options.LockTimeout, func() error {
		mapping, found, err := readHookMapping(pluginData, hostHash)
		if err != nil {
			return err
		}
		newSession := !found || input.EventName == "SessionStart" && (input.Source == "startup" || input.Source == "clear")
		if newSession {
			mapping, result, err = createHookSession(options, pluginData, hostHash, input)
			if err != nil {
				return err
			}
		} else {
			result.SessionID = mapping.AFRSessionID
			result.SessionDir, err = ResolveSessionRoot(options.SessionsRoot, mapping.AFRSessionID)
			if err != nil {
				return err
			}
		}
		state, err := appendCodexHook(options, mapping, result.SessionDir, input)
		result.State = state
		return err
	})
	return result, err
}

func parseCodexHook(input io.Reader) (codexHook, error) {
	if input == nil {
		return codexHook{}, errors.New("hook input is required")
	}
	data, err := io.ReadAll(io.LimitReader(input, maxHookInputBytes+1))
	if err != nil {
		return codexHook{}, fmt.Errorf("read hook input: %w", err)
	}
	if len(data) > maxHookInputBytes {
		return codexHook{}, fmt.Errorf("hook input exceeds %d bytes", maxHookInputBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	fields := map[string]json.RawMessage{}
	if err := decoder.Decode(&fields); err != nil {
		return codexHook{}, fmt.Errorf("decode hook input: %w", err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return codexHook{}, fmt.Errorf("decode hook input: %w", err)
	}
	readString := func(name string) (string, error) {
		raw, ok := fields[name]
		if !ok || bytes.Equal(raw, []byte("null")) {
			return "", nil
		}
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return "", fmt.Errorf("%s must be a string", name)
		}
		return value, nil
	}
	result := codexHook{Values: map[string]any{}}
	for name, target := range map[string]*string{
		"hook_event_name": &result.EventName,
		"session_id":      &result.SessionID,
		"turn_id":         &result.TurnID,
		"source":          &result.Source,
		"cwd":             &result.CWD,
		"tool_name":       &result.ToolName,
		"tool_use_id":     &result.ToolUseID,
		"permission_mode": &result.PermissionMode,
		"model":           &result.Model,
	} {
		*target, err = readString(name)
		if err != nil {
			return codexHook{}, err
		}
	}
	if !hostIDPattern.MatchString(result.SessionID) {
		return codexHook{}, errors.New("session_id has an invalid format")
	}
	if result.TurnID != "" && !hostIDPattern.MatchString(result.TurnID) {
		return codexHook{}, errors.New("turn_id has an invalid format")
	}
	allowedEvents := map[string]bool{"SessionStart": true, "UserPromptSubmit": true, "PreToolUse": true, "PostToolUse": true, "Stop": true}
	if !allowedEvents[result.EventName] {
		return codexHook{}, fmt.Errorf("unsupported Codex hook event %q", result.EventName)
	}
	if result.EventName == "SessionStart" {
		allowedSources := map[string]bool{"startup": true, "resume": true, "clear": true, "compact": true}
		if !allowedSources[result.Source] {
			return codexHook{}, fmt.Errorf("unsupported SessionStart source %q", result.Source)
		}
	}
	known := map[string]bool{
		"hook_event_name": true, "session_id": true, "turn_id": true, "source": true, "cwd": true,
		"tool_name": true, "tool_use_id": true, "permission_mode": true, "model": true,
		"prompt": true, "tool_input": true, "tool_response": true, "last_assistant_message": true,
		"transcript_path": true,
	}
	for _, name := range []string{"prompt", "tool_input", "tool_response", "last_assistant_message"} {
		if raw, ok := fields[name]; ok {
			var value any
			if err := json.Unmarshal(raw, &value); err != nil {
				return codexHook{}, fmt.Errorf("decode %s: %w", name, err)
			}
			result.Values[name] = value
		}
	}
	for name, raw := range fields {
		if known[name] {
			continue
		}
		sum := sha256.Sum256(raw)
		result.Omitted = append(result.Omitted, hookOmittedField{Name: name, Type: jsonType(raw), Bytes: len(raw), SHA256: hex.EncodeToString(sum[:]), Reason: "unknown_field"})
	}
	sort.Slice(result.Omitted, func(i, j int) bool { return result.Omitted[i].Name < result.Omitted[j].Name })
	return result, nil
}

func createHookSession(options HookOptions, pluginData, hostHash string, input codexHook) (hookMapping, HookResult, error) {
	workspace, err := resolveWorkspace(input.CWD)
	if err != nil {
		return hookMapping{}, HookResult{}, err
	}
	key := make([]byte, sha256.Size)
	if _, err := rand.Read(key); err != nil {
		return hookMapping{}, HookResult{}, err
	}
	fingerprinter, _ := NewFingerprinterFromKey(key)
	redactor, err := hookRedactor(workspace, fingerprinter)
	if err != nil {
		return hookMapping{}, HookResult{}, err
	}
	session, err := NewSession(options.SessionsRoot, workspace, []string{"codex-desktop"}, redactor)
	if err != nil {
		return hookMapping{}, HookResult{}, err
	}
	before := CollectWorkspace(workspace, options.SessionsRoot, fingerprinter, options.ScanLimits)
	session.Meta.CaptureMode = "desktop_hook"
	session.Meta.Host = "codex"
	session.Meta.HostSession = hostHash
	session.Meta.Capabilities = hookCapabilities(before, false)
	if err := WriteWorkspaceArtifact(session.Root, "workspace-before.json", before, redactor); err != nil {
		return hookMapping{}, HookResult{}, err
	}
	writer, err := NewEventWriter(session.Root, redactor)
	if err != nil {
		return hookMapping{}, HookResult{}, err
	}
	_, err = writer.Append("session_started", "codex", map[string]any{
		"capture_mode": "desktop_hook", "host": "codex", "host_session_fingerprint": hostHash,
		"workspace": workspace, "capabilities": session.Meta.Capabilities,
	}, true)
	seq, finalHash := writer.Snapshot()
	closeErr := writer.Close()
	if err != nil || closeErr != nil {
		return hookMapping{}, HookResult{}, errors.Join(err, closeErr)
	}
	if err := session.Checkpoint("active", seq, finalHash); err != nil {
		return hookMapping{}, HookResult{}, err
	}
	mapping := hookMapping{FormatVersion: 1, HostSessionHash: hostHash, AFRSessionID: session.Meta.ID, Workspace: workspace, Key: hex.EncodeToString(key)}
	if err := writeHookMapping(pluginData, mapping); err != nil {
		return hookMapping{}, HookResult{}, err
	}
	return mapping, HookResult{SessionID: session.Meta.ID, SessionDir: session.Root, State: "active"}, nil
}

func appendCodexHook(options HookOptions, mapping hookMapping, sessionRoot string, input codexHook) (string, error) {
	key, err := hex.DecodeString(mapping.Key)
	if err != nil {
		return "", errors.New("hook mapping key is invalid")
	}
	fingerprinter, err := NewFingerprinterFromKey(key)
	if err != nil {
		return "", err
	}
	redactor, err := hookRedactor(mapping.Workspace, fingerprinter)
	if err != nil {
		return "", err
	}
	metadata, issue := readSessionForVerify(sessionRoot)
	if issue != nil {
		return "", errors.New(issue.Message)
	}
	if input.CWD != "" {
		cwd, err := resolveWorkspace(input.CWD)
		if err != nil || !samePath(cwd, mapping.Workspace) {
			return "", errors.New("hook cwd differs from mapped workspace")
		}
	}
	started, _ := time.Parse(time.RFC3339Nano, metadata.StartedAt)
	writer, err := OpenEventWriter(sessionRoot, redactor, started)
	if err != nil {
		return "", err
	}
	payload := hookPayload(input)
	state := "active"
	findings := []RiskFinding{}
	if input.EventName == "Stop" {
		state, findings, err = finalizeHookTurn(options, &metadata, sessionRoot, writer, fingerprinter, redactor, payload)
	} else {
		eventType := map[string]string{"SessionStart": "host_session_started", "UserPromptSubmit": "prompt_submitted", "PreToolUse": "tool_started", "PostToolUse": "tool_finished"}[input.EventName]
		_, err = writer.Append(eventType, "codex", payload, true)
	}
	seq, finalHash := writer.Snapshot()
	counts := writer.Counts()
	closeErr := writer.Close()
	if err != nil || closeErr != nil {
		return "", errors.Join(err, closeErr)
	}
	session := &Session{Root: sessionRoot, Meta: metadata, redactor: redactor}
	if err := session.Checkpoint(state, seq, finalHash); err != nil {
		return "", err
	}
	if state == "idle" {
		if err := writeHookReport(session, counts, findings, redactor); err != nil {
			return "", err
		}
	}
	return state, nil
}

func finalizeHookTurn(options HookOptions, metadata *SessionMetadata, sessionRoot string, writer *EventWriter, fingerprinter *Fingerprinter, redactor *Redactor, stopPayload map[string]any) (string, []RiskFinding, error) {
	before, err := readWorkspaceSnapshot(sessionRoot, "workspace-before.json")
	if err != nil {
		return "", nil, err
	}
	after := CollectWorkspace(metadata.Workspace, options.SessionsRoot, fingerprinter, options.ScanLimits)
	counts := writer.Counts()
	metadata.Capabilities = hookCapabilities(after, counts["tool_started"] > 0 || counts["tool_finished"] > 0)
	if err := WriteWorkspaceArtifact(sessionRoot, "workspace-after.json", after, redactor); err != nil {
		return "", nil, err
	}
	delta := CompareWorkspace(before, after)
	patch, err := GenerateWorkspacePatch(metadata.Workspace, sessionRoot, before, after, delta, fingerprinter, redactor, 0)
	if err != nil {
		return "", nil, err
	}
	delta.Patch = patch
	if err := WriteWorkspaceArtifact(sessionRoot, "workspace-delta.json", delta, redactor); err != nil {
		return "", nil, err
	}
	workspaceSeq, err := writer.Append("workspace_final", "workspace", map[string]any{
		"added": len(delta.Added), "modified": len(delta.Modified), "deleted": len(delta.Deleted),
		"renamed": len(delta.Renamed), "pre_existing": len(delta.PreExisting),
	}, true)
	if err != nil {
		return "", nil, err
	}
	risks := NewRiskSet()
	if finding, found := BulkChangeRisk(delta, workspaceSeq); found {
		risks.Add(finding)
	}
	if finding, found := SensitiveRisk("workspace patch", patch.Redactions, workspaceSeq); found {
		risks.Add(finding)
	}
	findings := risks.Findings()
	for _, finding := range findings {
		if _, err := writer.Append("risk_found", "policy", finding, true); err != nil {
			return "", nil, err
		}
	}
	stopPayload["state"] = "idle"
	_, err = writer.Append("turn_stopped", "codex", stopPayload, true)
	return "idle", findings, err
}

func writeHookReport(session *Session, counts map[string]uint64, findings []RiskFinding, redactor *Redactor) error {
	delta, err := readWorkspaceDelta(session.Root)
	if err != nil {
		return err
	}
	timeline, err := readReportTimeline(session.Root)
	if err != nil {
		return err
	}
	findings = append(findings, timeline.risks...)
	view := NewReportView(session.Meta, delta, findings, counts, nil)
	view.Timeline = timeline
	if err := WriteReportArtifacts(session.Root, view, redactor); err != nil {
		return err
	}
	return WriteManifest(session.Root, session.Meta.ID, requiredEvidencePaths, derivedReportPaths, redactor)
}

func hookPayload(input codexHook) map[string]any {
	payload := map[string]any{"host_session_fingerprint": hashText(input.SessionID)}
	for name, value := range map[string]string{
		"turn_fingerprint": hashText(input.TurnID), "source": input.Source, "tool_name": input.ToolName,
		"tool_use_fingerprint": hashText(input.ToolUseID), "permission_mode": input.PermissionMode, "model": input.Model,
	} {
		if value != "" {
			payload[name] = value
		}
	}
	for name, value := range input.Values {
		payload[name] = value
	}
	if len(input.Omitted) > 0 {
		payload["omitted_fields"] = input.Omitted
	}
	return payload
}

func hookCapabilities(snapshot WorkspaceSnapshot, nativeToolEventsObserved bool) []string {
	git := "workspace_git=not_observable"
	if snapshot.Git.Available && snapshot.Git.Error == "" {
		git = "workspace_git=observed"
	}
	native := "native_tool_events=not_observable"
	if nativeToolEventsObserved {
		native = "native_tool_events=observed"
	}
	return []string{"process=not_observable", git, "workspace_scan=observed", native, "local_policy=observed", "os_file_monitor=not_observable", "network_monitor=not_observable"}
}

func hookRedactor(workspace string, fingerprinter *Fingerprinter) (*Redactor, error) {
	custom, err := loadWorkspaceRedactionRules(workspace)
	if err != nil {
		return nil, err
	}
	return newRedactor(fingerprinter, custom)
}

func readWorkspaceSnapshot(sessionRoot, name string) (WorkspaceSnapshot, error) {
	var snapshot WorkspaceSnapshot
	err := readHookJSON(sessionRoot, filepath.Join("snapshots", name), &snapshot)
	return snapshot, err
}

func readWorkspaceDelta(sessionRoot string) (WorkspaceDelta, error) {
	var delta WorkspaceDelta
	err := readHookJSON(sessionRoot, filepath.Join("snapshots", "workspace-delta.json"), &delta)
	return delta, err
}

func readHookJSON(root, relative string, destination any) error {
	path, err := safeJoin(root, relative)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(data) > maxHookArtifactBytes {
		return fmt.Errorf("%s exceeds read limit", relative)
	}
	return json.Unmarshal(data, destination)
}

func resolveHookRoot(root string) (string, error) {
	if root == "" {
		return "", errors.New("PLUGIN_DATA is required")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(absolute, 0o700); err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(absolute)
}

func readHookMapping(pluginData, hostHash string) (hookMapping, bool, error) {
	directory := filepath.Join(pluginData, "sessions")
	path := filepath.Join(directory, hostHash+".json")
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return hookMapping{}, false, nil
	}
	if err != nil {
		return hookMapping{}, false, err
	}
	if len(data) > 4096 {
		return hookMapping{}, false, errors.New("hook mapping exceeds limit")
	}
	var mapping hookMapping
	if err := json.Unmarshal(data, &mapping); err != nil || mapping.FormatVersion != 1 || mapping.HostSessionHash != hostHash || mapping.AFRSessionID == "" {
		return hookMapping{}, false, errors.New("hook mapping is invalid")
	}
	return mapping, true, nil
}

func writeHookMapping(pluginData string, mapping hookMapping) error {
	directory := filepath.Join(pluginData, "sessions")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(mapping)
	if err != nil {
		return err
	}
	return atomicWriteBytes(directory, mapping.HostSessionHash+".json", append(data, '\n'))
}

func withHookLock(pluginData, hostHash string, timeout time.Duration, action func() error) error {
	directory := filepath.Join(pluginData, "locks")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	path := filepath.Join(directory, hostHash+".lock")
	deadline := time.Now().Add(timeout)
	for {
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			owner, _ := json.Marshal(hookLockOwner{PID: os.Getpid(), CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)})
			_, writeErr := file.Write(owner)
			closeErr := file.Close()
			if writeErr != nil || closeErr != nil {
				_ = os.Remove(path)
				return errors.Join(writeErr, closeErr)
			}
			defer os.Remove(path)
			return action()
		}
		if !errors.Is(err, os.ErrExist) {
			return err
		}
		if staleHookLock(path) {
			_ = os.Remove(path)
			continue
		}
		if time.Now().After(deadline) {
			return errors.New("hook session lock timed out")
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func staleHookLock(path string) bool {
	info, err := os.Stat(path)
	if err != nil || time.Since(info.ModTime()) <= hookLockStale {
		return false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return true
	}
	var owner hookLockOwner
	return json.Unmarshal(data, &owner) != nil || !processAlive(owner.PID)
}

func hashText(value string) string {
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func jsonType(raw []byte) string {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return "invalid"
	}
	switch trimmed[0] {
	case '{':
		return "object"
	case '[':
		return "array"
	case '"':
		return "string"
	case 't', 'f':
		return "boolean"
	case 'n':
		return "null"
	default:
		return "number"
	}
}
