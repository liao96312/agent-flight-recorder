package afr

import (
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

const defaultOutputLimit = int64(200 * 1024 * 1024)

type RunOptions struct {
	SessionsRoot string
	Workspace    string
	Stdout       io.Writer
	Stderr       io.Writer
	Interrupts   <-chan os.Signal
	GracePeriod  time.Duration
	OutputLimit  int64
	ScanLimits   ScanLimits
	RecordLimit  int
}

type RunResult struct {
	SessionID  string
	SessionDir string
	ExitCode   int
}

type outputCapture struct {
	stream      string
	chunks      uint64
	totalBytes  int64
	recorded    int64
	fingerprint hash.Hash
	records     *RecordAccumulator
}

func Run(options RunOptions, argv []string) (RunResult, error) {
	if len(argv) == 0 {
		return RunResult{}, errors.New("empty command")
	}
	workspace, err := resolveWorkspace(options.Workspace)
	if err != nil {
		return RunResult{}, err
	}
	if options.SessionsRoot == "" {
		options.SessionsRoot, err = DefaultSessionsRoot()
		if err != nil {
			return RunResult{}, err
		}
	}
	if options.Stdout == nil {
		options.Stdout = io.Discard
	}
	if options.Stderr == nil {
		options.Stderr = io.Discard
	}
	if options.GracePeriod <= 0 {
		options.GracePeriod = 5 * time.Second
	}
	if options.OutputLimit <= 0 {
		options.OutputLimit = defaultOutputLimit
	}
	fingerprinter, err := NewFingerprinter()
	if err != nil {
		return RunResult{}, err
	}
	redactor, err := NewRedactor(fingerprinter)
	if err != nil {
		return RunResult{}, err
	}

	session, err := NewSession(options.SessionsRoot, workspace, argv, redactor)
	if err != nil {
		return RunResult{}, err
	}
	result := RunResult{SessionID: session.Meta.ID, SessionDir: session.Root}
	writer, err := NewEventWriter(session.Root, redactor)
	if err != nil {
		return result, err
	}
	riskSet := NewRiskSet()
	before := CollectWorkspace(workspace, options.SessionsRoot, fingerprinter, options.ScanLimits)
	session.Meta.Capabilities = WorkspaceCapabilities(before)
	redactedArgv, argvRedactions := redactor.ArgvWithKinds(argv)
	sessionStartedSeq, err := writer.Append("session_started", "afr", map[string]any{
		"session_id":        session.Meta.ID,
		"workspace":         workspace,
		"command":           filepath.Base(argv[0]),
		"arg_count":         len(argv) - 1,
		"argv_redacted":     redactedArgv,
		"environment_names": redactor.EnvironmentNames(os.Environ()),
		"capabilities":      session.Meta.Capabilities,
	}, true)
	if err != nil {
		_ = writer.Close()
		return result, err
	}
	if finding, found := CommandRisk(argv, sessionStartedSeq); found {
		riskSet.Add(finding)
	}
	if finding, found := SensitiveRisk("argv", argvRedactions, sessionStartedSeq); found {
		riskSet.Add(finding)
	}
	for _, finding := range ObservableBoundaryRisks(workspace, argv, sessionStartedSeq) {
		riskSet.Add(finding)
	}
	if err := WriteWorkspaceArtifact(session.Root, "workspace-before.json", before, redactor); err != nil {
		return finishSetupFailure(session, writer, result, "workspace_baseline", err)
	}
	if _, err := writer.Append("workspace_baseline", "workspace", map[string]any{
		"mode":         before.Mode,
		"partial":      before.Partial,
		"files":        len(before.Files),
		"pre_existing": len(before.Git.Entries),
		"artifact":     "snapshots/workspace-before.json",
	}, true); err != nil {
		_ = writer.Close()
		return result, err
	}

	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = workspace
	tree, err := prepareProcessTree(cmd)
	if err != nil {
		return finishSetupFailure(session, writer, result, "start_child", err)
	}
	defer tree.close()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return finishSetupFailure(session, writer, result, "start_child", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return finishSetupFailure(session, writer, result, "start_child", err)
	}
	started := time.Now()
	if err := cmd.Start(); err != nil {
		return finishSetupFailure(session, writer, result, "start_child", err)
	}
	if err := tree.afterStart(cmd); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return finishSetupFailure(session, writer, result, "start_child", err)
	}
	if _, err := writer.Append("process_started", "process", map[string]any{
		"pid":        cmd.Process.Pid,
		"executable": filepath.Base(argv[0]),
		"cwd":        workspace,
	}, true); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		_ = writer.Close()
		return result, err
	}

	stdoutCapture := outputCapture{stream: "stdout", fingerprint: fingerprinter.NewHash(), records: NewRecordAccumulator(redactor, options.RecordLimit)}
	stderrCapture := outputCapture{stream: "stderr", fingerprint: fingerprinter.NewHash(), records: NewRecordAccumulator(redactor, options.RecordLimit)}
	var pumps sync.WaitGroup
	var captureMu sync.Mutex
	var captureErr error
	pump := func(state *outputCapture, reader io.Reader, terminal io.Writer) {
		defer pumps.Done()
		buffer := make([]byte, 32*1024)
		persistRecord := func(record RedactedRecord) error {
			if state.recorded+int64(record.Bytes) > options.OutputLimit {
				return nil
			}
			state.recorded += int64(record.Bytes)
			state.chunks++
			payload := map[string]any{
				"stream":     state.stream,
				"record_seq": state.chunks,
				"bytes":      record.Bytes,
			}
			if record.OmittedReason == "" {
				payload["text"] = record.Text
			} else {
				payload["content_omitted"] = true
				payload["omitted_reason"] = record.OmittedReason
				payload["fingerprint"] = record.Fingerprint
			}
			seq, err := writer.Append("process_output", "process", payload, false)
			if err == nil {
				if finding, found := SensitiveRisk(state.stream, record.Redactions, seq); found {
					riskSet.Add(finding)
				}
			}
			return err
		}
		for {
			n, readErr := reader.Read(buffer)
			if n > 0 {
				state.totalBytes += int64(n)
				_, _ = state.fingerprint.Write(buffer[:n])
				written, terminalErr := terminal.Write(buffer[:n])
				if terminalErr == nil && written != n {
					terminalErr = io.ErrShortWrite
				}
				var eventErr error
				for _, record := range state.records.Feed(buffer[:n]) {
					if err := persistRecord(record); err != nil && eventErr == nil {
						eventErr = err
					}
				}
				if terminalErr != nil || eventErr != nil {
					captureMu.Lock()
					if captureErr == nil {
						captureErr = errors.Join(terminalErr, eventErr)
					}
					captureMu.Unlock()
				}
			}
			if readErr != nil {
				for _, record := range state.records.Close() {
					if err := persistRecord(record); err != nil {
						captureMu.Lock()
						if captureErr == nil {
							captureErr = err
						}
						captureMu.Unlock()
					}
				}
				if !errors.Is(readErr, io.EOF) {
					captureMu.Lock()
					if captureErr == nil {
						captureErr = readErr
					}
					captureMu.Unlock()
				}
				return
			}
		}
	}
	pumps.Add(2)
	go pump(&stdoutCapture, stdout, options.Stdout)
	go pump(&stderrCapture, stderr, options.Stderr)
	waitDone := make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()
	waitErr, interrupted, forced, signalName, forwardOK := waitForChild(cmd, tree, waitDone, options.Interrupts, options.GracePeriod)
	pumps.Wait()
	for _, capture := range []*outputCapture{&stdoutCapture, &stderrCapture} {
		if capture.totalBytes > options.OutputLimit {
			if _, err := writer.Append("output_truncated", "process", map[string]any{
				"stream":             capture.stream,
				"limit":              options.OutputLimit,
				"total_bytes":        capture.totalBytes,
				"stream_fingerprint": hex.EncodeToString(capture.fingerprint.Sum(nil)),
			}, true); err != nil {
				_ = writer.Close()
				return result, err
			}
		}
	}

	exitCode := 0
	termination := "exited"
	if waitErr != nil {
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			termination = "wait_failed"
		}
	}
	result.ExitCode = exitCode
	if interrupted {
		if _, err := writer.Append("session_interrupted", "afr", map[string]any{
			"signal":    signalName,
			"forwarded": forwardOK,
			"forced":    forced,
			"exit_code": exitCode,
		}, true); err != nil {
			_ = writer.Close()
			return result, err
		}
	}
	after := CollectWorkspace(workspace, options.SessionsRoot, fingerprinter, options.ScanLimits)
	delta := CompareWorkspace(before, after)
	patch, patchErr := GenerateWorkspacePatch(workspace, session.Root, before, after, delta, fingerprinter, redactor, defaultDiffLimit)
	if patchErr != nil {
		_ = writer.Close()
		return result, patchErr
	}
	delta.Patch = patch
	if patch.Error != "" {
		after.Git.Error = patch.Error
	}
	session.Meta.Capabilities = WorkspaceCapabilities(after)
	if err := WriteWorkspaceArtifact(session.Root, "workspace-after.json", after, redactor); err != nil {
		_ = writer.Close()
		return result, err
	}
	if err := WriteWorkspaceArtifact(session.Root, "workspace-delta.json", delta, redactor); err != nil {
		_ = writer.Close()
		return result, err
	}
	workspaceFinalSeq, err := writer.Append("workspace_final", "workspace", map[string]any{
		"mode":      after.Mode,
		"partial":   delta.Partial,
		"added":     len(delta.Added),
		"modified":  len(delta.Modified),
		"deleted":   len(delta.Deleted),
		"renamed":   len(delta.Renamed),
		"artifacts": []string{"snapshots/workspace-after.json", "snapshots/workspace-delta.json", "diffs/workspace.patch"},
	}, true)
	if err != nil {
		_ = writer.Close()
		return result, err
	}
	if finding, found := BulkChangeRisk(delta, workspaceFinalSeq); found {
		riskSet.Add(finding)
	}
	if finding, found := SensitiveRisk("workspace patch", patch.Redactions, workspaceFinalSeq); found {
		riskSet.Add(finding)
	}
	findings := riskSet.Findings()
	if _, err := writer.Append("process_exited", "process", map[string]any{
		"exit_code":   exitCode,
		"termination": termination,
		"duration_ns": time.Since(started).Nanoseconds(),
	}, true); err != nil {
		_ = writer.Close()
		return result, err
	}
	for _, finding := range findings {
		if _, err := writer.Append("risk_found", "policy", finding, true); err != nil {
			_ = writer.Close()
			return result, err
		}
	}
	if _, err := writer.Append("session_finished", "afr", map[string]any{
		"state":         "completed",
		"child_success": exitCode == 0,
		"risk_count":    len(findings),
		"highest_risk":  highestSeverity(findings),
	}, true); err != nil {
		_ = writer.Close()
		return result, err
	}
	seq, hash := writer.Snapshot()
	if err := writer.Close(); err != nil {
		return result, err
	}
	if err := session.Finish("completed", &exitCode, seq, hash); err != nil {
		return result, err
	}
	if err := WriteManifest(session.Root, session.Meta.ID, requiredEvidencePaths, nil, redactor); err != nil {
		return result, err
	}
	captureMu.Lock()
	err = captureErr
	captureMu.Unlock()
	if err != nil {
		return result, fmt.Errorf("capture child output: %w", err)
	}
	if termination == "wait_failed" {
		return result, fmt.Errorf("wait for child: %w", waitErr)
	}
	return result, nil
}

func waitForChild(command *exec.Cmd, tree *processTree, waitDone <-chan error, interrupts <-chan os.Signal, gracePeriod time.Duration) (waitErr error, interrupted, forced bool, signalName string, forwardOK bool) {
	if interrupts == nil {
		return <-waitDone, false, false, "", false
	}
	select {
	case waitErr = <-waitDone:
		return waitErr, false, false, "", false
	case received, ok := <-interrupts:
		if !ok {
			return <-waitDone, false, false, "", false
		}
		interrupted = true
		signalName = received.String()
		forwardOK = tree.interrupt(command) == nil
	}

	timer := time.NewTimer(gracePeriod)
	defer timer.Stop()
	select {
	case waitErr = <-waitDone:
		return waitErr, interrupted, false, signalName, forwardOK
	case _, ok := <-interrupts:
		if !ok {
			return <-waitDone, interrupted, false, signalName, forwardOK
		}
		forced = true
	case <-timer.C:
		forced = true
	}
	if err := tree.killCommand(command); err != nil {
		_ = command.Process.Kill()
	}
	waitErr = <-waitDone
	return waitErr, interrupted, forced, signalName, forwardOK
}

func resolveWorkspace(path string) (string, error) {
	if path == "" {
		var err error
		path, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("get workspace: %w", err)
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve workspace: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("inspect workspace: %w", err)
	}
	if !info.IsDir() {
		return "", errors.New("workspace is not a directory")
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("resolve workspace links: %w", err)
	}
	return resolved, nil
}

func finishSetupFailure(session *Session, writer *EventWriter, result RunResult, stage string, cause error) (RunResult, error) {
	_, _ = writer.Append("session_finished", "afr", map[string]any{"state": "failed", "stage": stage}, true)
	seq, hash := writer.Snapshot()
	closeErr := writer.Close()
	finishErr := session.Finish("failed", nil, seq, hash)
	return result, errors.Join(fmt.Errorf("start child: %w", cause), closeErr, finishErr)
}
