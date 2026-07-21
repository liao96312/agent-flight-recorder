package afr

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestReportViewIsSharedAndDeterministic(t *testing.T) {
	exitCode := 7
	session := SessionMetadata{ID: "session", State: "completed", ChildExitCode: &exitCode, FinalEventSeq: 9, Capabilities: []string{"workspace_scan=observed", "network_monitor=not_observable"}}
	changes := WorkspaceDelta{Added: []string{"z", "a"}, Modified: []string{}, Deleted: []string{}, Renamed: []RenameEvidence{}, PreExisting: []string{}, Partial: true}
	findings := []RiskFinding{
		{RuleID: "low", RuleVersion: 1, Severity: "low", EvidenceSeq: []uint64{8}},
		{RuleID: "high", RuleVersion: 1, Severity: "high", EvidenceSeq: []uint64{3}},
	}
	view := NewReportView(session, changes, findings, map[string]uint64{"session_finished": 1, "process_output": 4})
	if !reflect.DeepEqual(view.Events, []ReportEventCount{{Type: "process_output", Count: 4}, {Type: "session_finished", Count: 1}}) || !reflect.DeepEqual(view.Changes.Added, []string{"a", "z"}) || view.Risks[0].RuleID != "high" || view.HighestRisk != "high" {
		t.Fatalf("view=%+v", view)
	}

	root := t.TempDir()
	redactor, _ := NewRedactor(&Fingerprinter{})
	if err := WriteReportArtifacts(root, view, redactor); err != nil {
		t.Fatal(err)
	}
	markdown, err := os.ReadFile(filepath.Join(root, "agent-flight.md"))
	if err != nil || !strings.Contains(string(markdown), "Events: 9") || !strings.Contains(string(markdown), "`network_monitor=not_observable`") {
		t.Fatalf("markdown=%s error=%v", markdown, err)
	}
	riskJSON, err := os.ReadFile(filepath.Join(root, "agent-risk.json"))
	if err != nil {
		t.Fatal(err)
	}
	var risk RiskReport
	if err := json.Unmarshal(riskJSON, &risk); err != nil || risk.SessionID != "session" || risk.Findings[0].RuleID != "high" || !reflect.DeepEqual(risk.Limitations, view.Limitations) {
		t.Fatalf("risk=%+v error=%v", risk, err)
	}
}
