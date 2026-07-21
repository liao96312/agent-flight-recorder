package afr

import (
	"net"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const bulkChangeThreshold = 100

type RiskFinding struct {
	RuleID      string   `json:"rule_id"`
	RuleVersion int      `json:"rule_version"`
	Severity    string   `json:"severity"`
	EvidenceSeq []uint64 `json:"evidence_seq"`
	Observed    []string `json:"observed"`
	Inferred    string   `json:"inferred"`
	Explanation string   `json:"explanation"`
	Action      string   `json:"action"`
}

type RiskSet struct {
	mu       sync.Mutex
	findings map[string]RiskFinding
}

func NewRiskSet() *RiskSet { return &RiskSet{findings: map[string]RiskFinding{}} }

func (set *RiskSet) Add(finding RiskFinding) {
	set.mu.Lock()
	defer set.mu.Unlock()
	if current, exists := set.findings[finding.RuleID]; exists {
		current.EvidenceSeq = append(current.EvidenceSeq, finding.EvidenceSeq...)
		current.Observed = append(current.Observed, finding.Observed...)
		set.findings[finding.RuleID] = current
		return
	}
	set.findings[finding.RuleID] = finding
}

func (set *RiskSet) Findings() []RiskFinding {
	set.mu.Lock()
	defer set.mu.Unlock()
	findings := make([]RiskFinding, 0, len(set.findings))
	for _, finding := range set.findings {
		sort.Slice(finding.EvidenceSeq, func(i, j int) bool { return finding.EvidenceSeq[i] < finding.EvidenceSeq[j] })
		finding.EvidenceSeq = compactUint64(finding.EvidenceSeq)
		sort.Strings(finding.Observed)
		finding.Observed = compactStrings(finding.Observed)
		findings = append(findings, finding)
	}
	sort.Slice(findings, func(i, j int) bool {
		left, right := severityRank(findings[i].Severity), severityRank(findings[j].Severity)
		if left == right {
			return findings[i].RuleID < findings[j].RuleID
		}
		return left < right
	})
	return findings
}

func CommandRisk(argv []string, evidenceSeq uint64) (RiskFinding, bool) {
	action := destructiveAction(argv)
	if action == "" {
		return RiskFinding{}, false
	}
	return RiskFinding{
		RuleID:      "command.destructive",
		RuleVersion: 1,
		Severity:    "high",
		EvidenceSeq: []uint64{evidenceSeq},
		Observed:    []string{"visible argv matched destructive action: " + action},
		Inferred:    "the requested command can delete or irreversibly replace data",
		Explanation: "AFR matched only the command argv it directly observed; child-internal commands remain not observable.",
		Action:      "Confirm the target and ensure a recoverable backup exists.",
	}, true
}

func SensitiveRisk(source string, kinds []string, evidenceSeq uint64) (RiskFinding, bool) {
	if len(kinds) == 0 {
		return RiskFinding{}, false
	}
	sort.Strings(kinds)
	return RiskFinding{
		RuleID:      "content.sensitive",
		RuleVersion: 1,
		Severity:    "high",
		EvidenceSeq: []uint64{evidenceSeq},
		Observed:    []string{source + " contained redacted sensitive pattern(s): " + strings.Join(compactStrings(kinds), ",")},
		Inferred:    "sensitive material was exposed to the recorded process or its visible output",
		Explanation: "The finding references redacted evidence only; AFR does not restore or copy the matched value.",
		Action:      "Rotate exposed credentials when appropriate and avoid passing secrets through argv or output.",
	}, true
}

func BulkChangeRisk(delta WorkspaceDelta, evidenceSeq uint64) (RiskFinding, bool) {
	count := len(delta.Added) + len(delta.Modified) + len(delta.Deleted) + len(delta.Renamed)
	if count < bulkChangeThreshold {
		return RiskFinding{}, false
	}
	return RiskFinding{
		RuleID:      "workspace.bulk_change",
		RuleVersion: 1,
		Severity:    "medium",
		EvidenceSeq: []uint64{evidenceSeq},
		Observed: []string{
			"session delta files: " + strconv.Itoa(count),
			"scope added=" + strconv.Itoa(len(delta.Added)) + ", modified=" + strconv.Itoa(len(delta.Modified)) + ", deleted=" + strconv.Itoa(len(delta.Deleted)) + ", renamed=" + strconv.Itoa(len(delta.Renamed)),
		},
		Inferred:    "the session changed an unusually large workspace scope",
		Explanation: "Rule v1 triggers at 100 session-delta files; pre-existing changes are excluded.",
		Action:      "Review the workspace patch and file summary before accepting the changes.",
	}, true
}

func ObservableBoundaryRisks(workspace string, argv []string, evidenceSeq uint64) []RiskFinding {
	findings := []RiskFinding{}
	if paths := outsideWorkspacePaths(workspace, argv); len(paths) > 0 {
		findings = append(findings, RiskFinding{
			RuleID:      "boundary.path_outside_workspace",
			RuleVersion: 1,
			Severity:    "medium",
			EvidenceSeq: []uint64{evidenceSeq},
			Observed:    paths,
			Inferred:    "the visible command may access a path outside the declared workspace",
			Explanation: "AFR matched explicit argv paths only; OS-level file activity remains not observable.",
			Action:      "Confirm each outside-workspace path is intentional.",
		})
	}
	if elevation := elevationAction(argv); elevation != "" {
		findings = append(findings, RiskFinding{
			RuleID:      "permission.elevation_requested",
			RuleVersion: 1,
			Severity:    "high",
			EvidenceSeq: []uint64{evidenceSeq},
			Observed:    []string{"visible argv requested elevation through: " + elevation},
			Inferred:    "the command requested execution with elevated permissions",
			Explanation: "AFR observed the request syntax, not whether elevation was granted or used.",
			Action:      "Verify elevation is necessary and limit the elevated command scope.",
		})
	}
	if target := networkTarget(argv); target != "" {
		findings = append(findings, RiskFinding{
			RuleID:      "network.target_visible",
			RuleVersion: 1,
			Severity:    "medium",
			EvidenceSeq: []uint64{evidenceSeq},
			Observed:    []string{"visible network target: " + target},
			Inferred:    "the command requested a network-capable operation",
			Explanation: "AFR observed a domain or IP in argv; actual network traffic remains not observable.",
			Action:      "Confirm the destination and data scope are expected.",
		})
	}
	return findings
}

func outsideWorkspacePaths(workspace string, argv []string) []string {
	paths := []string{}
	for _, argument := range argv[1:] {
		candidate := argument
		if _, value, found := strings.Cut(candidate, "="); found {
			candidate = value
		}
		if strings.Contains(candidate, "://") {
			continue
		}
		isParentRelative := strings.HasPrefix(candidate, ".."+string(filepath.Separator)) || strings.HasPrefix(candidate, "../") || strings.HasPrefix(candidate, `..\`)
		if !filepath.IsAbs(candidate) && !isParentRelative {
			continue
		}
		resolved := candidate
		if !filepath.IsAbs(resolved) {
			resolved = filepath.Join(workspace, resolved)
		}
		resolved, err := filepath.Abs(resolved)
		if err != nil {
			continue
		}
		if linked, err := filepath.EvalSymlinks(resolved); err == nil {
			resolved = linked
		}
		if !pathWithin(workspace, resolved) {
			paths = append(paths, "visible argv path outside workspace: "+resolved)
		}
	}
	sort.Strings(paths)
	return compactStrings(paths)
}

func elevationAction(argv []string) string {
	if len(argv) == 0 {
		return ""
	}
	command := commandName(argv[0])
	switch command {
	case "sudo", "doas", "pkexec", "runas":
		return command
	case "powershell", "pwsh":
		for index, argument := range argv[1:] {
			if !hasAny([]string{argument}, "-command", "-c") || index+2 > len(argv)-1 {
				continue
			}
			script := strings.ToLower(strings.TrimSpace(argv[index+2]))
			if strings.HasPrefix(script, "start-process ") && strings.Contains(script, "-verb runas") {
				return "powershell Start-Process -Verb RunAs"
			}
		}
	}
	return ""
}

func networkTarget(argv []string) string {
	if len(argv) < 2 {
		return ""
	}
	command := commandName(argv[0])
	args := argv[1:]
	if command == "git" {
		if len(args) < 2 || !hasAny([]string{args[0]}, "clone", "fetch", "pull", "push", "ls-remote") {
			return ""
		}
	} else if command == "powershell" || command == "pwsh" {
		for index, argument := range args {
			if hasAny([]string{argument}, "-command", "-c") && index+1 < len(args) {
				script := strings.Fields(args[index+1])
				if len(script) > 1 && hasAny([]string{script[0]}, "Invoke-WebRequest", "Invoke-RestMethod") {
					return firstNetworkTarget(script[1:], false)
				}
			}
		}
		return ""
	} else if !hasAny([]string{command}, "curl", "wget", "scp", "sftp", "ftp", "ssh", "rsync", "ping", "nslookup") {
		return ""
	}
	allowBareHost := hasAny([]string{command}, "ping", "nslookup", "ssh", "sftp", "ftp")
	return firstNetworkTarget(args, allowBareHost)
}

func firstNetworkTarget(args []string, allowBareHost bool) string {
	for _, argument := range args {
		candidate := strings.TrimSpace(argument)
		if candidate == "" || strings.HasPrefix(candidate, "-") {
			continue
		}
		if _, value, found := strings.Cut(candidate, "="); found {
			candidate = value
		}
		if parsed, err := url.Parse(candidate); err == nil && parsed.Scheme != "" && parsed.Hostname() != "" {
			return strings.ToLower(parsed.Hostname())
		}
		remoteSyntax := strings.Contains(candidate, "@") || strings.Contains(candidate, ":") && !(len(candidate) >= 2 && candidate[1] == ':')
		if remoteSyntax {
			if at := strings.LastIndex(candidate, "@"); at >= 0 {
				candidate = candidate[at+1:]
			}
			if colon := strings.Index(candidate, ":"); colon > 0 {
				candidate = candidate[:colon]
			}
		}
		candidate = strings.Trim(candidate, "[]")
		if (remoteSyntax || allowBareHost) && validNetworkHost(candidate) {
			return strings.ToLower(candidate)
		}
	}
	return ""
}

func validNetworkHost(host string) bool {
	if net.ParseIP(host) != nil {
		return true
	}
	if strings.EqualFold(host, "localhost") || !strings.Contains(host, ".") || strings.ContainsAny(host, `/\`) {
		return false
	}
	labels := strings.Split(host, ".")
	for _, label := range labels {
		if label == "" || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
		for _, character := range label {
			if character < 'a' || character > 'z' {
				if character < 'A' || character > 'Z' {
					if character < '0' || character > '9' {
						if character != '-' {
							return false
						}
					}
				}
			}
		}
	}
	return true
}

func commandName(path string) string {
	return strings.ToLower(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
}

func destructiveAction(argv []string) string {
	if len(argv) == 0 {
		return ""
	}
	command := commandName(argv[0])
	args := argv[1:]
	switch command {
	case "rm", "rmdir", "del", "erase", "remove-item", "clear-content", "shred", "format", "diskpart":
		if hasOperand(args) && !hasAny(args, "--help", "--version", "-whatif") {
			return command
		}
	case "git":
		if len(args) >= 2 && strings.EqualFold(args[0], "reset") && hasAny(args[1:], "--hard") {
			return "git reset --hard"
		}
		if len(args) >= 2 && strings.EqualFold(args[0], "clean") && hasForce(args[1:]) && !hasDryRun(args[1:]) {
			return "git clean --force"
		}
	case "powershell", "pwsh":
		return destructiveShellAction(args, []string{"-command", "-c"}, []string{"remove-item ", "clear-content ", "format-volume "}, "-whatif")
	case "cmd":
		return destructiveShellAction(args, []string{"/c"}, []string{"del ", "erase ", "rmdir ", "rd ", "format "}, "")
	case "sh", "bash", "zsh":
		return destructiveShellAction(args, []string{"-c"}, []string{"rm ", "rmdir ", "shred "}, "")
	}
	return ""
}

func destructiveShellAction(args, switches, prefixes []string, safeMarker string) string {
	for index, argument := range args {
		if !hasAny([]string{argument}, switches...) || index+1 >= len(args) {
			continue
		}
		input := strings.ToLower(strings.TrimSpace(args[index+1]))
		if safeMarker != "" && strings.Contains(input, safeMarker) {
			return ""
		}
		for _, prefix := range prefixes {
			if input == strings.TrimSpace(prefix) || strings.HasPrefix(input, prefix) {
				return strings.TrimSpace(prefix)
			}
		}
	}
	return ""
}

func hasOperand(args []string) bool {
	for _, argument := range args {
		if argument != "" && (!strings.HasPrefix(argument, "-") || argument == "-") {
			return true
		}
	}
	return false
}

func hasAny(args []string, values ...string) bool {
	for _, argument := range args {
		for _, value := range values {
			if strings.EqualFold(argument, value) {
				return true
			}
		}
	}
	return false
}

func hasForce(args []string) bool {
	for _, argument := range args {
		lower := strings.ToLower(argument)
		if lower == "--force" || strings.HasPrefix(lower, "-") && !strings.HasPrefix(lower, "--") && strings.Contains(lower[1:], "f") {
			return true
		}
	}
	return false
}

func hasDryRun(args []string) bool {
	for _, argument := range args {
		lower := strings.ToLower(argument)
		if lower == "--dry-run" || lower == "-n" || strings.HasPrefix(lower, "-") && !strings.HasPrefix(lower, "--") && strings.Contains(lower[1:], "n") {
			return true
		}
	}
	return false
}

func severityRank(severity string) int {
	for rank, value := range []string{"critical", "high", "medium", "low", "info"} {
		if severity == value {
			return rank
		}
	}
	return 5
}

func highestSeverity(findings []RiskFinding) string {
	if len(findings) == 0 {
		return "none"
	}
	return findings[0].Severity
}

func compactUint64(values []uint64) []uint64 {
	if len(values) == 0 {
		return values
	}
	result := values[:1]
	for _, value := range values[1:] {
		if value != result[len(result)-1] {
			result = append(result, value)
		}
	}
	return result
}
