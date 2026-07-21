package afr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestReportViewIsSharedAndDeterministic(t *testing.T) {
	exitCode := 7
	session := SessionMetadata{ID: "session", State: "completed", ChildExitCode: &exitCode, FinalEventSeq: 9, Capabilities: []string{"workspace_scan=observed", "network_monitor=not_observable"}}
	changes := WorkspaceDelta{Added: []string{"z", "a"}, Modified: []string{}, Deleted: []string{}, Renamed: []RenameEvidence{}, PreExisting: []string{}, Partial: true}
	findings := []RiskFinding{
		{RuleID: "low", RuleVersion: 1, Severity: "low", EvidenceSeq: []uint64{8}},
		{RuleID: "high", RuleVersion: 1, Severity: "high", EvidenceSeq: []uint64{3}},
	}
	view := NewReportView(session, changes, findings, map[string]uint64{"session_finished": 1, "process_output": 4}, nil)
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

func TestHTMLReportEscapesInjectionAndBoundsContent(t *testing.T) {
	payload := `</script><img src="https://example.invalid/leak">`
	added := make([]string, 1001)
	for index := range added {
		added[index] = payload + string(rune('a'+index%26))
	}
	view := NewReportView(
		SessionMetadata{ID: "safe-session", State: "completed"},
		WorkspaceDelta{Added: added, Modified: []string{}, Deleted: []string{}, Renamed: []RenameEvidence{}, PreExisting: []string{}},
		[]RiskFinding{{RuleID: "injection", RuleVersion: 1, Severity: "high", Explanation: payload, Action: payload}},
		map[string]uint64{"process_output": 1},
		[]ReportStream{{Stream: "stdout", Preview: strings.Repeat("界", 5000), TotalBytes: 15000, Fingerprint: strings.Repeat("a", 64)}},
	)
	root := t.TempDir()
	redactor, _ := NewRedactor(&Fingerprinter{})
	if err := writeHTMLReport(root, view, redactor); err != nil {
		t.Fatal(err)
	}
	document, err := os.ReadFile(filepath.Join(root, "report.html"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(document, []byte(payload)) || bytes.Contains(document, []byte("<img")) || !bytes.Contains(document, []byte("&lt;/script&gt;")) {
		t.Fatalf("unsafe HTML output: %s", document)
	}
	decoded := html.UnescapeString(string(document))
	for _, expected := range []string{contentHash(reportScript), contentHash(reportStyle), `id="filter"`, "1 more paths omitted", "Preview is limited to 4 KiB"} {
		if !strings.Contains(decoded, expected) {
			t.Fatalf("HTML missing %q", expected)
		}
	}
	if len(view.Output[0].Preview) > reportPreviewLimit || bytes.Count(document, []byte("界")) > reportPreviewLimit/len([]byte("界"))+1 {
		t.Fatal("HTML output preview exceeded its byte limit")
	}
}

func TestReportTimelineBoundsHundredThousandEvents(t *testing.T) {
	root := t.TempDir()
	redactor, _ := NewRedactor(&Fingerprinter{})
	writer, err := NewEventWriter(root, redactor)
	if err != nil {
		t.Fatal(err)
	}
	for index := 1; index <= 100000; index++ {
		payload := map[string]any{"index": index}
		if index == 1 {
			payload["text"] = strings.Repeat("界", 300)
		}
		if _, err := writer.Append("fixture", "test", payload, false); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	timeline, err := readReportTimeline(root)
	if err != nil {
		t.Fatal(err)
	}
	if timeline.TotalEvents != 100000 || len(timeline.Items) != 1001 {
		t.Fatalf("timeline total=%d items=%d", timeline.TotalEvents, len(timeline.Items))
	}
	marker := timeline.Items[500]
	if timeline.Items[0].Seq != 1 || timeline.Items[499].Seq != 500 || marker.OmittedCount != 99000 || marker.OmittedFrom != 501 || marker.OmittedTo != 99500 || timeline.Items[501].Seq != 99501 || timeline.Items[1000].Seq != 100000 {
		t.Fatalf("unexpected timeline bounds: first=%d marker=%+v last-first=%d last=%d", timeline.Items[0].Seq, marker, timeline.Items[501].Seq, timeline.Items[1000].Seq)
	}
	if !timeline.Items[0].SummaryTruncated || len(timeline.Items[0].Summary) > reportTimelineSummaryLimit || !utf8.ValidString(timeline.Items[0].Summary) {
		t.Fatalf("summary was not safely bounded: bytes=%d truncated=%t", len(timeline.Items[0].Summary), timeline.Items[0].SummaryTruncated)
	}

	view := NewReportView(SessionMetadata{ID: "timeline", State: "completed"}, WorkspaceDelta{}, nil, map[string]uint64{"fixture": 100000}, nil)
	view.Timeline = timeline
	if err := writeHTMLReport(root, view, redactor); err != nil {
		t.Fatal(err)
	}
	document, err := os.ReadFile(filepath.Join(root, "report.html"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Count(document, []byte("data-event-row")) != 1000 || !bytes.Contains(document, []byte("99000 events omitted (seq 501–99500)")) || !bytes.Contains(document, []byte("[summary truncated]")) {
		t.Fatal("HTML timeline did not preserve its fixed bounds and markers")
	}
}

func TestFormatRunSummaryUsesCapabilityVector(t *testing.T) {
	exitCode := 7
	view := NewReportView(
		SessionMetadata{ID: "session", State: "completed", ChildExitCode: &exitCode, Capabilities: []string{"workspace_scan=observed", "network_monitor=not_observable", "process=observed", "native_tool_events=not_observable"}},
		WorkspaceDelta{Added: []string{"a"}, Modified: []string{"b", "c"}, Deleted: []string{"d"}, Renamed: []RenameEvidence{{From: "e", To: "f"}}, PreExisting: []string{"g"}},
		[]RiskFinding{{RuleID: "risk", RuleVersion: 1, Severity: "high"}},
		nil,
		[]ReportStream{{Stream: "stdout", Truncated: true}},
	)
	want := fmt.Sprintf("Session: session\nEvidence: observed=process,workspace_scan; not_observable=native_tool_events,network_monitor; truncated=true\nChanged: added=1 modified=2 deleted=1 renamed=1 pre_existing=1\nRisks: total=1 highest=high\nExit: child=7 state=completed\nReport: %s\n", "/tmp/report.html")
	if got := FormatRunSummary(view, "/tmp/report.html"); got != want {
		t.Fatalf("summary mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}
