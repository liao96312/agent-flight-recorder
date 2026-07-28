package afr

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

var derivedReportPaths = []string{"agent-flight.md", "agent-risk.json", "report.html"}

const reportPreviewLimit = 4 * 1024
const reportTimelineEdgeLimit = 500
const reportTimelineSummaryLimit = 512

type ReportEventCount struct {
	Type  string `json:"type"`
	Count uint64 `json:"count"`
}

type ReportTimelineItem struct {
	Seq              uint64 `json:"seq,omitempty"`
	Timestamp        string `json:"timestamp,omitempty"`
	Type             string `json:"type,omitempty"`
	Source           string `json:"source,omitempty"`
	Summary          string `json:"summary,omitempty"`
	SummaryTruncated bool   `json:"summary_truncated,omitempty"`
	OmittedCount     uint64 `json:"omitted_count,omitempty"`
	OmittedFrom      uint64 `json:"omitted_from,omitempty"`
	OmittedTo        uint64 `json:"omitted_to,omitempty"`
}

type ReportTimeline struct {
	TotalEvents uint64               `json:"total_events"`
	Items       []ReportTimelineItem `json:"items"`
	risks       []RiskFinding
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
	Timeline      ReportTimeline     `json:"timeline"`
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
	if counts["tool_started"] != counts["tool_finished"] {
		limitations = append(limitations, "native_tool_outcomes=partial")
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

func readReportTimeline(sessionRoot string) (ReportTimeline, error) {
	path, err := safeJoin(sessionRoot, "events.jsonl")
	if err != nil {
		return ReportTimeline{}, err
	}
	file, err := os.Open(path)
	if err != nil {
		return ReportTimeline{}, fmt.Errorf("open report timeline: %w", err)
	}
	defer file.Close()

	first := make([]ReportTimelineItem, 0, reportTimelineEdgeLimit)
	last := make([]ReportTimelineItem, reportTimelineEdgeLimit)
	lastCount, nextLast := 0, 0
	total := uint64(0)
	risks := NewRiskSet()
	reader := bufio.NewReaderSize(file, 64*1024)
	for {
		line, readErr, tooLarge := readBoundedEventLine(reader)
		if tooLarge {
			return ReportTimeline{}, errors.New("event exceeds report timeline line limit")
		}
		if readErr == io.EOF && len(line) > 0 {
			return ReportTimeline{}, errors.New("events file has a torn final record")
		}
		if len(line) > 0 {
			var envelope eventEnvelope
			if err := json.Unmarshal(bytes.TrimSuffix(line, []byte{'\n'}), &envelope); err != nil {
				return ReportTimeline{}, fmt.Errorf("decode report timeline envelope: %w", err)
			}
			var body EventBody
			if err := json.Unmarshal(envelope.Body, &body); err != nil {
				return ReportTimeline{}, fmt.Errorf("decode report timeline event: %w", err)
			}
			if body.Type == "risk_found" {
				var finding RiskFinding
				if err := json.Unmarshal(body.Payload, &finding); err != nil {
					return ReportTimeline{}, fmt.Errorf("decode report risk: %w", err)
				}
				risks.Add(finding)
			}
			summary := string(body.Payload)
			item := ReportTimelineItem{
				Seq:              body.Seq,
				Timestamp:        body.Timestamp,
				Type:             body.Type,
				Source:           body.Source,
				Summary:          truncateUTF8(summary, reportTimelineSummaryLimit),
				SummaryTruncated: len(summary) > reportTimelineSummaryLimit,
			}
			total++
			if len(first) < reportTimelineEdgeLimit {
				first = append(first, item)
			} else if lastCount < reportTimelineEdgeLimit {
				last[lastCount] = item
				lastCount++
			} else {
				last[nextLast] = item
				nextLast = (nextLast + 1) % reportTimelineEdgeLimit
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return ReportTimeline{}, fmt.Errorf("read report timeline: %w", readErr)
		}
	}

	orderedLast := make([]ReportTimelineItem, 0, lastCount)
	for index := range lastCount {
		orderedLast = append(orderedLast, last[(nextLast+index)%reportTimelineEdgeLimit])
	}
	items := append([]ReportTimelineItem{}, first...)
	omitted := total - uint64(len(first)+lastCount)
	if omitted > 0 {
		items = append(items, ReportTimelineItem{
			OmittedCount: omitted,
			OmittedFrom:  first[len(first)-1].Seq + 1,
			OmittedTo:    orderedLast[0].Seq - 1,
		})
	}
	items = append(items, orderedLast...)
	return ReportTimeline{TotalEvents: total, Items: items, risks: risks.Findings()}, nil
}

func FormatRunSummary(view ReportView, reportPath string) string {
	observed, notObservable := []string{}, []string{}
	for _, capability := range view.Session.Capabilities {
		name, state, ok := strings.Cut(capability, "=")
		if !ok {
			continue
		}
		switch state {
		case "observed":
			observed = append(observed, name)
		case "not_observable":
			notObservable = append(notObservable, name)
		}
	}
	sort.Strings(observed)
	sort.Strings(notObservable)
	truncated := view.Changes.Partial || view.Changes.Patch.Truncated
	for _, stream := range view.Output {
		truncated = truncated || stream.Truncated
	}
	join := func(values []string) string {
		if len(values) == 0 {
			return "none"
		}
		return strings.Join(values, ",")
	}
	return fmt.Sprintf(
		"Session: %s\nEvidence: observed=%s; not_observable=%s; truncated=%t\nChanged: added=%d modified=%d deleted=%d renamed=%d pre_existing=%d\nRisks: total=%d highest=%s\nExit: child=%s state=%s\nReport: %s\n",
		view.Session.ID,
		join(observed),
		join(notObservable),
		truncated,
		len(view.Changes.Added),
		len(view.Changes.Modified),
		len(view.Changes.Deleted),
		len(view.Changes.Renamed),
		len(view.Changes.PreExisting),
		len(view.Risks),
		view.HighestRisk,
		exitCodeText(view.Session.ChildExitCode),
		view.Session.State,
		reportPath,
	)
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
