package afr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRunCapturesNonGitDelta(t *testing.T) {
	if os.Getenv("AFR_TEST_WORKSPACE_CHILD") == "1" {
		root := os.Getenv("AFR_TEST_WORKSPACE_ROOT")
		_ = os.WriteFile(filepath.Join(root, "modify.txt"), []byte("after"), 0o600)
		_ = os.Rename(filepath.Join(root, "rename.txt"), filepath.Join(root, "renamed.txt"))
		_ = os.WriteFile(filepath.Join(root, "added.txt"), []byte("new"), 0o600)
		os.Exit(0)
	}
	workspace := t.TempDir()
	for name, content := range map[string]string{"modify.txt": "before", "rename.txt": "same"} {
		if err := os.WriteFile(filepath.Join(workspace, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("AFR_TEST_WORKSPACE_CHILD", "1")
	t.Setenv("AFR_TEST_WORKSPACE_ROOT", workspace)
	result, err := Run(RunOptions{SessionsRoot: filepath.Join(t.TempDir(), "sessions"), Workspace: workspace}, []string{os.Args[0], "-test.run=TestRunCapturesNonGitDelta"})
	if err != nil {
		t.Fatal(err)
	}
	delta := readDelta(t, result.SessionDir)
	if !reflect.DeepEqual(delta.Added, []string{"added.txt"}) || !reflect.DeepEqual(delta.Modified, []string{"modify.txt"}) || !reflect.DeepEqual(delta.Renamed, []RenameEvidence{{From: "rename.txt", To: "renamed.txt"}}) || delta.Partial {
		t.Fatalf("delta=%+v", delta)
	}
	if verification := VerifySession(result.SessionDir); !verification.Valid {
		t.Fatalf("verification=%+v", verification)
	}
}

func TestRunSeparatesGitPreExistingChanges(t *testing.T) {
	if os.Getenv("AFR_TEST_GIT_CHILD") == "1" {
		root := os.Getenv("AFR_TEST_GIT_ROOT")
		_ = os.WriteFile(filepath.Join(root, "changed.txt"), []byte("session"), 0o600)
		_ = os.WriteFile(filepath.Join(root, "added.txt"), []byte("session"), 0o600)
		os.Exit(0)
	}
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git unavailable")
	}
	workspace := t.TempDir()
	runGitFixture(t, git, workspace, "init", "-q")
	for _, name := range []string{"preexisting.txt", "changed.txt"} {
		if err := os.WriteFile(filepath.Join(workspace, name), []byte("base"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	runGitFixture(t, git, workspace, "add", ".")
	runGitFixture(t, git, workspace, "commit", "-qm", "base")
	if err := os.WriteFile(filepath.Join(workspace, "preexisting.txt"), []byte("dirty before"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AFR_TEST_GIT_CHILD", "1")
	t.Setenv("AFR_TEST_GIT_ROOT", workspace)
	result, err := Run(RunOptions{SessionsRoot: filepath.Join(t.TempDir(), "sessions"), Workspace: workspace}, []string{os.Args[0], "-test.run=TestRunSeparatesGitPreExistingChanges"})
	if err != nil {
		t.Fatal(err)
	}
	delta := readDelta(t, result.SessionDir)
	if !reflect.DeepEqual(delta.Added, []string{"added.txt"}) || !reflect.DeepEqual(delta.Modified, []string{"changed.txt"}) || !reflect.DeepEqual(delta.PreExisting, []string{"preexisting.txt"}) {
		t.Fatalf("delta=%+v", delta)
	}
}

func TestRunOutputLimitDrainsAndFingerprints(t *testing.T) {
	if os.Getenv("AFR_TEST_OUTPUT_LIMIT") == "1" {
		_, _ = os.Stdout.Write(bytes.Repeat([]byte("secret"), 1024))
		os.Exit(0)
	}
	t.Setenv("AFR_TEST_OUTPUT_LIMIT", "1")
	output := &countWriter{}
	result, err := Run(RunOptions{
		SessionsRoot: filepath.Join(t.TempDir(), "sessions"),
		Workspace:    t.TempDir(),
		Stdout:       output,
		OutputLimit:  64,
	}, []string{os.Args[0], "-test.run=TestRunOutputLimitDrainsAndFingerprints"})
	if err != nil || output.bytes != 6*1024 {
		t.Fatalf("result=%+v bytes=%d error=%v", result, output.bytes, err)
	}
	events, err := os.ReadFile(filepath.Join(result.SessionDir, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(events, []byte("secret")) {
		t.Fatal("truncated raw output reached evidence")
	}
	found := false
	for _, line := range bytes.Split(bytes.TrimSpace(events), []byte{'\n'}) {
		var envelope eventEnvelope
		if json.Unmarshal(line, &envelope) != nil {
			continue
		}
		var body EventBody
		if json.Unmarshal(envelope.Body, &body) != nil || body.Type != "output_truncated" {
			continue
		}
		var payload struct {
			Limit       int64  `json:"limit"`
			Total       int64  `json:"total_bytes"`
			Fingerprint string `json:"stream_fingerprint"`
		}
		if json.Unmarshal(body.Payload, &payload) != nil || payload.Limit != 64 || payload.Total != 6*1024 || len(payload.Fingerprint) != 64 {
			t.Fatalf("payload=%s", body.Payload)
		}
		found = true
	}
	if !found {
		t.Fatal("output_truncated event missing")
	}
}

func TestWorkspaceSymlinkOutsideIsNotFollowed(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "outside.txt"), []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	fingerprinter, _ := NewFingerprinter()
	snapshot := CollectWorkspace(root, "", fingerprinter, ScanLimits{})
	if len(snapshot.Files) != 1 || snapshot.Files[0].Type != "symlink" || !snapshot.Files[0].TargetOutside {
		t.Fatalf("snapshot=%+v", snapshot)
	}
}

func TestRunLeavesNoPlaintextSecretInSession(t *testing.T) {
	const secret = "supersecretvalue"
	if os.Getenv("AFR_TEST_SECRET_CHILD") == "1" {
		foundArg := false
		for _, argument := range os.Args {
			if argument == secret {
				foundArg = true
			}
		}
		if !foundArg {
			os.Exit(9)
		}
		fmt.Fprintln(os.Stdout, "password="+secret)
		fmt.Fprintln(os.Stderr, "Bearer "+secret+"abcdefghijkl")
		root := os.Getenv("AFR_TEST_SECRET_ROOT")
		_ = os.WriteFile(filepath.Join(root, "token="+secret+".txt"), []byte(secret), 0o600)
		os.Exit(0)
	}
	base := t.TempDir()
	workspace := filepath.Join(base, "password="+secret)
	if err := os.Mkdir(workspace, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AFR_TEST_SECRET_CHILD", "1")
	t.Setenv("AFR_TEST_SECRET_ROOT", workspace)
	t.Setenv("AFR_SECRET_ENV", secret)
	result, err := Run(RunOptions{
		SessionsRoot: filepath.Join(t.TempDir(), "sessions"),
		Workspace:    workspace,
	}, []string{os.Args[0], "-test.run=TestRunLeavesNoPlaintextSecretInSession", "--", "--token", secret})
	if err != nil {
		t.Fatal(err)
	}
	foundPlaceholder := false
	foundSensitiveRisk := false
	err = filepath.Walk(result.SessionDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info.IsDir() {
			return walkErr
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if bytes.Contains(data, []byte(secret)) {
			t.Errorf("plaintext secret in %s", path)
		}
		if bytes.Contains(data, []byte("[REDACTED:")) {
			foundPlaceholder = true
		}
		if bytes.Contains(data, []byte(`"type":"risk_found"`)) && bytes.Contains(data, []byte(`"rule_id":"content.sensitive"`)) {
			foundSensitiveRisk = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !foundPlaceholder {
		t.Fatal("redaction placeholder missing from session")
	}
	if !foundSensitiveRisk {
		t.Fatal("sensitive evidence did not produce a redacted risk finding")
	}
}

func readDelta(t *testing.T, sessionRoot string) WorkspaceDelta {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(sessionRoot, "snapshots", "workspace-delta.json"))
	if err != nil {
		t.Fatal(err)
	}
	var delta WorkspaceDelta
	if err := json.Unmarshal(data, &delta); err != nil {
		t.Fatal(err)
	}
	return delta
}

func runGitFixture(t *testing.T, git, root string, args ...string) {
	t.Helper()
	command := exec.Command(git, append([]string{"-C", root}, args...)...)
	command.Env = append(os.Environ(), "GIT_AUTHOR_NAME=AFR", "GIT_AUTHOR_EMAIL=afr@example.invalid", "GIT_COMMITTER_NAME=AFR", "GIT_COMMITTER_EMAIL=afr@example.invalid")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}
