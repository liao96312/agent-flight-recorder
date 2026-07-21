package afr

import (
	"path/filepath"
	"reflect"
	"slices"
	"testing"
)

func TestCommandRiskMatchesActionsNotQuotedExamples(t *testing.T) {
	risky := [][]string{
		{"rm", "-rf", "/tmp/work"},
		{"git", "reset", "--hard"},
		{"git", "clean", "-fd"},
		{"powershell", "-Command", "Remove-Item C:\\work -Recurse"},
		{"cmd", "/c", "del file.txt"},
		{"sh", "-c", "rm -rf ./build"},
	}
	for _, argv := range risky {
		if finding, found := CommandRisk(argv, 7); !found || finding.RuleID != "command.destructive" || !reflect.DeepEqual(finding.EvidenceSeq, []uint64{7}) {
			t.Fatalf("risk missing for %v: %+v", argv, finding)
		}
	}
	for _, argv := range [][]string{
		{"echo", "rm -rf /"},
		{"git", "clean", "-nfd"},
		{"git", "reset", "--soft"},
		{"powershell", "-Command", "Remove-Item C:\\work -WhatIf"},
		{"sh", "-c", "echo rm -rf /"},
		{"rm", "--help"},
	} {
		if finding, found := CommandRisk(argv, 7); found {
			t.Fatalf("false positive for %v: %+v", argv, finding)
		}
	}
}

func TestRiskSetMergesAndSortsFindings(t *testing.T) {
	set := NewRiskSet()
	first, _ := SensitiveRisk("stdout", []string{"bearer"}, 9)
	second, _ := SensitiveRisk("argv", []string{"argv"}, 2)
	bulk, _ := BulkChangeRisk(WorkspaceDelta{Added: make([]string, bulkChangeThreshold)}, 10)
	set.Add(bulk)
	set.Add(first)
	set.Add(second)
	findings := set.Findings()
	if len(findings) != 2 || findings[0].RuleID != "content.sensitive" || !reflect.DeepEqual(findings[0].EvidenceSeq, []uint64{2, 9}) || findings[1].RuleID != "workspace.bulk_change" || highestSeverity(findings) != "high" {
		t.Fatalf("findings=%+v", findings)
	}
}

func TestBulkChangeRiskUsesSessionDeltaThreshold(t *testing.T) {
	if _, found := BulkChangeRisk(WorkspaceDelta{Added: make([]string, bulkChangeThreshold-1), PreExisting: make([]string, 1000)}, 3); found {
		t.Fatal("pre-existing paths counted toward bulk rule")
	}
	finding, found := BulkChangeRisk(WorkspaceDelta{Added: make([]string, bulkChangeThreshold)}, 3)
	if !found || finding.RuleVersion != 1 || finding.Severity != "medium" {
		t.Fatalf("finding=%+v", finding)
	}
}

func TestObservableBoundaryRulesRequireVisibleTargets(t *testing.T) {
	workspace := t.TempDir()
	outside := filepath.Join(t.TempDir(), "target.txt")
	findings := ObservableBoundaryRisks(workspace, []string{"tool", "--output=" + outside}, 11)
	if len(findings) != 1 || findings[0].RuleID != "boundary.path_outside_workspace" || !reflect.DeepEqual(findings[0].EvidenceSeq, []uint64{11}) {
		t.Fatalf("outside findings=%+v", findings)
	}
	inside := filepath.Join(workspace, "target.txt")
	if findings := ObservableBoundaryRisks(workspace, []string{"tool", inside}, 11); len(findings) != 0 {
		t.Fatalf("inside path flagged: %+v", findings)
	}

	findings = ObservableBoundaryRisks(workspace, []string{"sudo", "tool"}, 12)
	if len(findings) != 1 || findings[0].RuleID != "permission.elevation_requested" {
		t.Fatalf("elevation findings=%+v", findings)
	}
	findings = ObservableBoundaryRisks(workspace, []string{"curl", "https://user:pass@example.invalid/upload"}, 13)
	if len(findings) != 1 || findings[0].RuleID != "network.target_visible" || !slices.Contains(findings[0].Observed, "visible network target: example.invalid") {
		t.Fatalf("network findings=%+v", findings)
	}
	if findings := ObservableBoundaryRisks(workspace, []string{"echo", "https://example.invalid"}, 14); len(findings) != 0 {
		t.Fatalf("quoted URL flagged as network activity: %+v", findings)
	}
}

func TestCapabilitiesKeepSystemMonitoringNotObservable(t *testing.T) {
	capabilities := WorkspaceCapabilities(WorkspaceSnapshot{})
	for _, expected := range []string{"local_policy=observed", "os_file_monitor=not_observable", "network_monitor=not_observable"} {
		if !slices.Contains(capabilities, expected) {
			t.Fatalf("missing %q in %v", expected, capabilities)
		}
	}
}
