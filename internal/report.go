package afr

import (
	"fmt"
	"sort"
	"strings"
)

var derivedReportPaths = []string{"agent-flight.md", "agent-risk.json", "report.html"}

const reportPreviewLimit = 4 * 1024

type ReportEventCount struct {
	Type  string `json:"type"`
	Count uint64 `json:"count"`
}

type ReportStream struct {
	Stream        string `json:"stream"`
	TotalBytes    int64  `json:"total_bytes"`
	RecordedBytes int64  `json:"recorded_bytes"`
	Records       uint64 `json:"records"`
	Truncated     bool   `json:"truncated"`
	Fingerprint   string `json:"fingerprint"`
	Preview       string `json:"preview,omitempty"`
}

type ReportView struct {
	FormatVersion int                `json:"format_version"`
	Session       SessionMetadata    `json:"session"`
	Events        []ReportEventCount `json:"events"`
	Output        []ReportStream     `json:"output"`
	Changes       WorkspaceDelta     `json:"changes"`
	Risks         []RiskFinding      `json:"risks"`
	HighestRisk   string             `json:"highest_risk"`
	Limitations   []string           `json:"limitations"`
}

type RiskReport struct {
	FormatVersion int           `json:"format_version"`
	SessionID     string        `json:"session_id"`
	State         string        `json:"state"`
	HighestRisk   string        `json:"highest_risk"`
	Findings      []RiskFinding `json:"findings"`
	Limitations   []string      `json:"limitations"`
}

func NewReportView(session SessionMetadata, changes WorkspaceDelta, findings []RiskFinding, counts map[string]uint64, output []ReportStream) ReportView {
	sort.Strings(session.Capabilities)
	sort.Strings(changes.Added)
	sort.Strings(changes.Modified)
	sort.Strings(changes.Deleted)
	sort.Strings(changes.PreExisting)
	sort.Slice(changes.Renamed, func(i, j int) bool {
		if changes.Renamed[i].From == changes.Renamed[j].From {
			return changes.Renamed[i].To < changes.Renamed[j].To
		}
		return changes.Renamed[i].From < changes.Renamed[j].From
	})
	riskSet := NewRiskSet()
	for _, finding := range findings {
		riskSet.Add(finding)
	}
	findings = riskSet.Findings()
	events := make([]ReportEventCount, 0, len(counts))
	for eventType, count := range counts {
		events = append(events, ReportEventCount{Type: eventType, Count: count})
	}
	sort.Slice(events, func(i, j int) bool { return events[i].Type < events[j].Type })
	sort.Slice(output, func(i, j int) bool { return output[i].Stream < output[j].Stream })
	for index := range output {
		output[index].Preview = truncateUTF8(output[index].Preview, reportPreviewLimit)
	}
	limitations := []string{}
	for _, capability := range session.Capabilities {
		if strings.HasSuffix(capability, "=not_observable") {
			limitations = append(limitations, capability)
		}
	}
	if changes.Partial {
		limitations = append(limitations, "workspace=partial")
	}
	if changes.Patch.Truncated {
		limitations = append(limitations, "workspace_patch=truncated")
	}
	limitations = append(limitations, changes.OmissionReasons...)
	sort.Strings(limitations)
	limitations = compactStrings(limitations)
	return ReportView{
		FormatVersion: 1,
		Session:       session,
		Events:        events,
		Output:        output,
		Changes:       changes,
		Risks:         findings,
		HighestRisk:   highestSeverity(findings),
		Limitations:   limitations,
	}
}

func truncateUTF8(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	for limit > 0 && value[limit]&0xc0 == 0x80 {
		limit--
	}
	return value[:limit]
}

func WriteReportArtifacts(sessionRoot string, view ReportView, redactor *Redactor) error {
	if err := atomicWriteRedactedText(sessionRoot, "agent-flight.md", markdownReport(view), redactor); err != nil {
		return err
	}
	risk := RiskReport{
		FormatVersion: 1,
		SessionID:     view.Session.ID,
		State:         view.Session.State,
		HighestRisk:   view.HighestRisk,
		Findings:      view.Risks,
		Limitations:   view.Limitations,
	}
	if err := atomicWriteJSON(sessionRoot, "agent-risk.json", risk, redactor); err != nil {
		return err
	}
	return writeHTMLReport(sessionRoot, view, redactor)
}

func markdownReport(view ReportView) string {
	var report strings.Builder
	fmt.Fprintf(&report, "# Agent Flight Report\n\n- Session: `%s`\n- State: **%s**\n- Child exit: %s\n- Events: %d\n- Highest risk: **%s**\n- Integrity: `afr verify %s`\n\n", view.Session.ID, view.Session.State, exitCodeText(view.Session.ChildExitCode), view.Session.FinalEventSeq, view.HighestRisk, view.Session.ID)
	fmt.Fprintln(&report, "## Capabilities")
	for _, capability := range view.Session.Capabilities {
		fmt.Fprintf(&report, "\n- `%s`", capability)
	}
	fmt.Fprintln(&report, "\n\n## Workspace changes")
	fmt.Fprintf(&report, "\n- Added: %d\n- Modified: %d\n- Deleted: %d\n- Renamed: %d\n- Pre-existing: %d\n", len(view.Changes.Added), len(view.Changes.Modified), len(view.Changes.Deleted), len(view.Changes.Renamed), len(view.Changes.PreExisting))
	fmt.Fprintln(&report, "\n## Risks")
	if len(view.Risks) == 0 {
		fmt.Fprintln(&report, "\nNo deterministic risk findings.")
	}
	for _, finding := range view.Risks {
		fmt.Fprintf(&report, "\n- **%s** `%s`: %s\n", finding.Severity, finding.RuleID, finding.Explanation)
	}
	if len(view.Limitations) > 0 {
		fmt.Fprintln(&report, "\n## Limitations")
		for _, limitation := range view.Limitations {
			fmt.Fprintf(&report, "\n- `%s`", limitation)
		}
		fmt.Fprintln(&report)
	}
	return report.String()
}

func exitCodeText(exitCode *int) string {
	if exitCode == nil {
		return "not available"
	}
	return fmt.Sprint(*exitCode)
}
