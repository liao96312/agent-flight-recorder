package afr

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunAppliesWorkspaceCustomRedaction(t *testing.T) {
	if os.Getenv("AFR_TEST_CUSTOM_REDACTION") == "1" {
		fmt.Fprint(os.Stdout, "ACME-1234")
		return
	}
	t.Setenv("AFR_TEST_CUSTOM_REDACTION", "1")
	root := t.TempDir()
	writeWorkspaceConfig(t, root, `{"redaction_rules":[{"name":"acme_id","type":"regex","pattern":"ACME-[0-9]{4}"}]}`)
	result, err := Run(RunOptions{Workspace: root, SessionsRoot: filepath.Join(t.TempDir(), "sessions")}, []string{os.Args[0], "-test.run=TestRunAppliesWorkspaceCustomRedaction"})
	if err != nil {
		t.Fatal(err)
	}
	events, err := os.ReadFile(filepath.Join(result.SessionDir, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(events), "ACME-1234") || !strings.Contains(string(events), "[REDACTED:acme_id:") {
		t.Fatalf("events=%s", events)
	}
}

func TestWorkspaceCustomRedactionRule(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceConfig(t, root, `{
  "redaction_rules": [
    {"name": "acme_id", "type": "regex", "pattern": "ACME-[0-9]{4}"}
  ]
}`)
	detectors, err := loadWorkspaceRedactionRules(root)
	if err != nil {
		t.Fatal(err)
	}
	fingerprinter, _ := NewFingerprinter()
	redactor, err := newRedactor(fingerprinter, detectors)
	if err != nil {
		t.Fatal(err)
	}
	redacted, kinds := redactor.TextWithKinds("keep ACME-123; hide ACME-1234")
	if !strings.Contains(redacted, "ACME-123") || strings.Contains(redacted, "ACME-1234") || len(kinds) != 1 || kinds[0] != "acme_id" {
		t.Fatalf("redacted=%q kinds=%v", redacted, kinds)
	}
}

func TestWorkspaceConfigRejectsInvalidRulesBeforeChildStart(t *testing.T) {
	tests := map[string]string{
		"invalid JSON":       `{`,
		"unknown root field": `{"unknown": true}`,
		"unknown rule field": `{"redaction_rules":[{"name":"custom","type":"regex","pattern":"x","unknown":true}]}`,
		"duplicate name":     `{"redaction_rules":[{"name":"custom","type":"regex","pattern":"x"},{"name":"custom","type":"regex","pattern":"y"}]}`,
		"built-in name":      `{"redaction_rules":[{"name":"bearer","type":"regex","pattern":"x"}]}`,
		"invalid type":       `{"redaction_rules":[{"name":"custom","type":"literal","pattern":"x"}]}`,
		"invalid regex":      `{"redaction_rules":[{"name":"custom","type":"regex","pattern":"["}]}`,
		"invalid name":       `{"redaction_rules":[{"name":"Custom Rule","type":"regex","pattern":"x"}]}`,
		"missing rules":      `{}`,
		"null rules":         `{"redaction_rules":null}`,
		"empty match":        `{"redaction_rules":[{"name":"custom","type":"regex","pattern":"a*"}]}`,
		"trailing value":     `{"redaction_rules":[]} {}`,
	}
	for name, config := range tests {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeWorkspaceConfig(t, root, config)
			result, err := Run(RunOptions{Workspace: root}, []string{"child-must-not-start"})
			if err == nil || result.SessionDir != "" {
				t.Fatalf("result=%+v error=%v", result, err)
			}
		})
	}
}

func writeWorkspaceConfig(t *testing.T, root, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, ".afr.json"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
