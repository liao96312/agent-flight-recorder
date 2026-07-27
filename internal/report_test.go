package afr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
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

func TestReportMarksPartialNativeToolOutcomes(t *testing.T) {
	view := NewReportView(
		SessionMetadata{Capabilities: []string{"native_tool_events=observed"}},
		WorkspaceDelta{},
		nil,
		map[string]uint64{"tool_started": 2, "tool_finished": 1},
		nil,
	)
	if !reflect.DeepEqual(view.Limitations, []string{"native_tool_outcomes=partial"}) {
		t.Fatalf("limitations = %v", view.Limitations)
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
	for _, expected := range []string{contentHash(reportScript), contentHash(reportStyle), `id="filter"`, "另有 1 条本次变化路径未在 HTML 中列出", "每个 stream 的预览最多 4 KiB"} {
		if !strings.Contains(decoded, expected) {
			t.Fatalf("HTML missing %q", expected)
		}
	}
	decodedDocument := html.UnescapeString(string(document))
	previewStart := strings.Index(decodedDocument, "<pre>")
	previewEnd := strings.Index(decodedDocument, "</pre>")
	if len(view.Output[0].Preview) > reportPreviewLimit || previewStart < 0 || previewEnd < previewStart || len([]byte(decodedDocument[previewStart+len("<pre>"):previewEnd])) > reportPreviewLimit {
		t.Fatal("HTML output preview exceeded its byte limit")
	}
}

func TestHTMLReportRedactsDataWithoutChangingCSPAssets(t *testing.T) {
	view := NewReportView(
		SessionMetadata{ID: "ACME-1234", State: "completed", Workspace: "ACME-1234"},
		WorkspaceDelta{Added: []string{"ACME-1234.txt"}},
		nil,
		nil,
		nil,
	)
	redactor, err := newRedactor(&Fingerprinter{}, []secretDetector{{
		name:    "custom",
		pattern: regexp.MustCompile(`system-ui|script-src|ACME-[0-9]+`),
	}})
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := writeHTMLReport(root, view, redactor); err != nil {
		t.Fatal(err)
	}
	document, err := os.ReadFile(filepath.Join(root, "report.html"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(document)
	for _, expected := range []string{"system-ui", "script-src", contentHash(reportScript), contentHash(reportStyle), "[REDACTED:custom:"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("HTML missing %q", expected)
		}
	}
	if strings.Contains(text, "ACME-1234") {
		t.Fatal("HTML retained unredacted report data")
	}
}

func TestReportStateLabelsDoNotOverclaimCompletion(t *testing.T) {
	tests := []struct {
		state          string
		label          string
		needsAttention bool
	}{
		{state: "idle", label: "上一轮已记录，等待下一轮"},
		{state: "starting", label: "正在记录当前任务"},
		{state: "running", label: "正在记录当前任务"},
		{state: "finalizing", label: "正在记录当前任务"},
		{state: "completed", label: "任务已结束并保存"},
		{state: "failed", label: "任务执行失败；本报告已保存，请检查失败原因", needsAttention: true},
		{state: "interrupted", label: "任务被中断，请检查记录完整性", needsAttention: true},
		{state: "incomplete", label: "记录不完整，需要检查", needsAttention: true},
		{state: "future", label: "未知状态：future", needsAttention: true},
	}
	for _, test := range tests {
		t.Run(test.state, func(t *testing.T) {
			got := reportState(test.state)
			if got.Label != test.label || got.NeedsAttention != test.needsAttention {
				t.Fatalf("state %q = %+v", test.state, got)
			}
		})
	}
}

func TestHTMLReportPutsHumanDecisionSummaryBeforeTechnicalEvidence(t *testing.T) {
	view := NewReportView(
		SessionMetadata{
			ID:          "desktop-session",
			State:       "idle",
			Workspace:   `D:\ai project`,
			Host:        "codex",
			CaptureMode: "desktop_hook",
			Capabilities: []string{
				"future_capability=unknown",
				"local_policy=observed",
				"native_tool_events=observed",
				"network_monitor=not_observable",
				"os_file_monitor=not_observable",
				"workspace_git=observed",
				"workspace_scan=observed",
			},
		},
		WorkspaceDelta{
			Added:           []string{"new.txt"},
			Modified:        []string{"changed.txt"},
			PreExisting:     []string{"before.txt"},
			Partial:         true,
			OmissionReasons: []string{"binary_content_omitted"},
			Patch: WorkspacePatchSummary{
				Truncated: true,
				Omitted:   []PatchOmission{{Path: "large.bin", Reason: "binary"}},
			},
		},
		nil,
		map[string]uint64{"tool_started": 2, "tool_finished": 1},
		[]ReportStream{{Stream: "stdout", Truncated: true}},
	)
	document := renderHTMLForTest(t, view)
	for _, expected := range []string{
		`lang="zh-CN"`,
		"上一轮已记录，等待下一轮",
		"未发现规则可确认的风险",
		"AFR 只判断已记录到的证据，这不代表绝对安全。",
		"开始记录前已存在，不归因于本次 AI。",
		"实际网络流量",
		"future_capability（状态未知）",
		"未监控表示 AFR 没有这类证据，不能据此判断该行为没有发生。",
		"项目文件扫描不完整",
		"工作区补丁已截断",
		"部分工具缺少结束事件，无法确认其执行结果",
		"部分二进制内容未进入补丁",
		"stdout 输出仅保留了一部分",
	} {
		if !strings.Contains(document, expected) {
			t.Fatalf("HTML missing %q", expected)
		}
	}

	ordered := []string{`<header class="hero">`, `id="attention"`, `id="changes"`, `id="evidence-boundary"`, `id="technical-details"`, "<summary>事件时间线</summary>"}
	last := -1
	for _, marker := range ordered {
		index := strings.Index(document, marker)
		if index <= last {
			t.Fatalf("%q was not in decision-first order: index=%d last=%d", marker, index, last)
		}
		last = index
	}
	if raw := strings.Index(document, "network_monitor=not_observable"); raw < strings.Index(document, "技术详情") {
		t.Fatalf("raw capability appeared before technical details: index=%d", raw)
	}
	if strings.Contains(document[:strings.Index(document, "技术详情")], "desktop-session") {
		t.Fatal("session ID appeared in the decision summary")
	}
}

func TestHTMLReportShowsActionableRiskAndIncompleteState(t *testing.T) {
	view := NewReportView(
		SessionMetadata{ID: "risk-session", State: "incomplete", Capabilities: []string{"workspace_scan=observed"}},
		WorkspaceDelta{},
		[]RiskFinding{{RuleID: "bulk-change", RuleVersion: 1, Severity: "high", Explanation: "检测到大量变化", Action: "先审查文件列表"}},
		nil,
		nil,
	)
	document := renderHTMLForTest(t, view)
	for _, expected := range []string{"记录不完整，需要检查", "发现 1 项需要检查，最高：高风险", "先审查文件列表", "全部风险与建议"} {
		if !strings.Contains(document, expected) {
			t.Fatalf("HTML missing %q", expected)
		}
	}
	if strings.Contains(document, "未发现规则可确认的风险") {
		t.Fatal("risk report rendered the no-findings conclusion")
	}
}

func TestReportCapabilitiesKeepAllObservedAndUnknownTruth(t *testing.T) {
	observed, missing := reportCapabilities([]string{"workspace_scan=observed", "local_policy=observed"})
	if len(observed) != 2 || len(missing) != 0 {
		t.Fatalf("observed=%+v missing=%+v", observed, missing)
	}
	observed, missing = reportCapabilities([]string{"future_capability=unknown"})
	if len(observed) != 0 || len(missing) != 1 || missing[0].Label != "future_capability（状态未知）" {
		t.Fatalf("observed=%+v missing=%+v", observed, missing)
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
	if bytes.Count(document, []byte("data-event-row")) != 1000 || !bytes.Contains(document, []byte("HTML 省略 99000 条事件（seq 501–99500）")) || !bytes.Contains(document, []byte("[摘要已截断]")) {
		t.Fatal("HTML timeline did not preserve its fixed bounds and markers")
	}
}

func renderHTMLForTest(t *testing.T, view ReportView) string {
	t.Helper()
	root := t.TempDir()
	redactor, err := NewRedactor(&Fingerprinter{})
	if err != nil {
		t.Fatal(err)
	}
	if err := writeHTMLReport(root, view, redactor); err != nil {
		t.Fatal(err)
	}
	document, err := os.ReadFile(filepath.Join(root, "report.html"))
	if err != nil {
		t.Fatal(err)
	}
	return html.UnescapeString(string(document))
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
