package afr

import (
	"reflect"
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
