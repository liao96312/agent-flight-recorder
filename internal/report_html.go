package afr

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
)

const reportStyle = `:root{color-scheme:light;--ink:#172033;--muted:#5b6878;--line:#dbe2ea;--paper:#fff;--canvas:#f3f5f8;--accent:#2457d6;--good:#17633a;--warn:#8a5200;--danger:#a40000}*{box-sizing:border-box}body{margin:0;background:var(--canvas);color:var(--ink);font:15px/1.45 system-ui,-apple-system,"Segoe UI",sans-serif}body>header,main{width:min(1180px,calc(100% - 32px));margin-inline:auto}.hero{margin-top:24px;padding:24px;border:1px solid var(--line);border-radius:18px;background:linear-gradient(135deg,#fff,#f8faff);box-shadow:0 12px 32px rgba(31,47,75,.07)}.eyebrow{margin:0 0 8px;color:var(--muted);font-size:.88rem}.title-row{display:flex;align-items:flex-start;justify-content:space-between;gap:16px}.title-row h1{margin:0;font-size:clamp(1.65rem,3vw,2.35rem);line-height:1.15}.state{margin:.45rem 0 0;color:var(--muted)}.badge{display:inline-flex;max-width:28rem;padding:.45rem .72rem;border:1px solid currentColor;border-radius:999px;font-weight:700;text-align:center}.badge.active{color:var(--accent);background:#eef4ff}.badge.good{color:var(--good);background:#edf8f1}.badge.warn{color:var(--warn);background:#fff7e8}.badge.neutral{color:#46566a;background:#f0f3f6}.verdict{display:grid;gap:3px;margin:18px 0;padding:13px 15px;border-left:4px solid var(--accent);border-radius:8px;background:#f2f6ff}.verdict.danger{border-color:var(--danger);background:#fff1f1}.verdict strong{font-size:1.08rem}.verdict span{color:var(--muted)}.metrics{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:10px}.metric{padding:12px;border:1px solid var(--line);border-radius:10px;background:var(--paper)}.metric b{display:block;font-size:1.45rem;line-height:1.1}.metric span{color:var(--muted);font-size:.86rem}main{padding:14px 0 32px}.decision-grid{display:grid;grid-template-columns:.9fr 1.15fr 1.1fr;gap:14px;align-items:start}.panel{margin:0 0 14px;padding:18px;border:1px solid var(--line);border-radius:14px;background:var(--paper)}.panel h2{margin:0 0 10px;font-size:1.08rem}.panel h3{margin:.85rem 0 .35rem;font-size:.96rem}.panel p{margin:.45rem 0}.muted{color:var(--muted)}.callout{padding:10px;border-radius:8px;background:#fff7e8;color:#623b00}.callout.danger{background:#fff0f0;color:#780000}.compact-list{margin:.5rem 0 0;padding-left:1.2rem}.compact-list li{margin:.28rem 0;overflow-wrap:anywhere}.file-kind{display:inline-block;min-width:3.5rem;color:var(--muted);font-size:.82rem}.boundary{padding:9px 10px;border-radius:8px;background:#f4f7fa}.boundary.missing{background:#fff7e8}.boundary ul{margin:.4rem 0 0;padding-left:1.15rem}.boundary-note{margin-top:10px!important;color:#654000;font-size:.88rem}.risk{padding:12px 0;border-top:1px solid var(--line)}.risk:first-of-type{border-top:0}.risk h3{margin:0}.critical,.high{color:var(--danger)}.medium{color:var(--warn)}.low{color:#46566a}details.panel{padding:0}details.panel>summary{cursor:pointer;padding:16px 18px;font-weight:750;list-style-position:inside}details.panel[open]>summary{border-bottom:1px solid var(--line)}.details-body{padding:16px 18px}.technical details{margin:12px 0;border:1px solid var(--line);border-radius:9px}.technical details>summary{padding:10px 12px;font-weight:650;cursor:pointer}.technical details>div{padding:0 12px 12px}.metadata{display:grid;grid-template-columns:max-content 1fr;gap:5px 14px}.metadata dt{color:var(--muted)}.metadata dd{margin:0;overflow-wrap:anywhere}label{display:block;margin-bottom:6px;font-weight:650}input{width:100%;padding:.68rem;border:1px solid #9aa8b8;border-radius:8px;font:inherit}.table-wrap{overflow-x:auto}table{width:100%;border-collapse:collapse}th,td{text-align:left;padding:.5rem;border-bottom:1px solid #e6ebf0;vertical-align:top}th{white-space:nowrap}code,pre{border-radius:5px;background:#eef2f6}code{padding:.12rem .28rem;overflow-wrap:anywhere}pre{max-height:28rem;padding:.75rem;overflow:auto;white-space:pre-wrap}.technical table{min-width:680px}[hidden]{display:none!important}@media(max-width:850px){.decision-grid{grid-template-columns:1fr}.title-row{align-items:flex-start}.metadata{grid-template-columns:1fr}.metadata dd{margin-bottom:7px}}@media(max-width:600px){body>header,main{width:min(100% - 20px,1180px)}.hero{margin-top:10px;padding:16px}.title-row{display:block}.badge{margin-top:12px}.metrics{grid-template-columns:1fr}.panel{padding:15px}.technical table{min-width:620px}}`

const reportScript = `(()=>{const q=document.getElementById('filter');if(!q)return;const apply=()=>{const value=q.value.toLowerCase();document.querySelectorAll('[data-filter-item]').forEach(item=>item.hidden=!item.textContent.toLowerCase().includes(value))};q.addEventListener('input',apply)})();`

const reportTemplate = `<!doctype html>
<html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<meta http-equiv="Content-Security-Policy" content="{{.CSP}}"><title>AFR · {{.Project}}</title><style>{{.Style}}</style></head>
<body><header class="hero"><p class="eyebrow">Agent Flight Recorder · {{.Project}} · {{.Host}}</p>
<div class="title-row"><div><h1>本次记录概览</h1><p class="state">{{.State.Label}}</p></div><span class="badge {{.State.Tone}}">{{.State.Badge}}</span></div>
{{if .HasTopRisk}}<div class="verdict danger"><strong>发现 {{len .View.Risks}} 项需要检查，最高：{{.HighestRiskLabel}}</strong><span>{{.TopRisk.Explanation}}{{if .TopRisk.Action}} 建议：{{.TopRisk.Action}}{{end}}</span></div>{{else}}<div class="verdict"><strong>未发现规则可确认的风险</strong><span>AFR 只判断已记录到的证据，这不代表绝对安全。</span></div>{{end}}
<div class="metrics"><div class="metric"><b>{{.AttentionCount}}</b><span>需检查</span></div><div class="metric"><b>{{.ChangeCount}}</b><span>本轮文件变化</span></div><div class="metric"><b>{{len .View.Changes.PreExisting}}</b><span>记录前已有改动</span></div><div class="metric"><b>{{.EvidenceGapCount}}</b><span>证据缺口</span></div></div></header>
<main><div class="decision-grid">
<section class="panel" id="attention"><h2>你需要检查</h2>{{if .HasAttention}}{{if .State.NeedsAttention}}<p class="callout danger"><strong>记录状态：</strong>{{.State.Label}}</p>{{end}}{{if .HasTopRisk}}<p class="callout"><strong>{{severityLabel .TopRisk.Severity}} · {{.TopRisk.RuleID}}</strong><br>{{.TopRisk.Action}}</p>{{end}}{{if .EvidenceIssues}}<p class="callout"><strong>证据有 {{len .EvidenceIssues}} 项降级。</strong>详情见“证据边界”。</p>{{end}}{{else}}<p>当前没有需要立即处理的确定性问题。</p><p class="muted">仍需结合右侧证据边界判断覆盖范围。</p>{{end}}</section>
<section class="panel" id="changes"><h2>本次改了什么</h2><p><strong>{{.ChangeCount}} 个文件变化</strong>：新增 {{len .View.Changes.Added}}、修改 {{len .View.Changes.Modified}}、删除 {{len .View.Changes.Deleted}}、重命名 {{len .View.Changes.Renamed}}。</p>{{if .ChangePreview}}<ul class="compact-list">{{range .ChangePreview}}<li data-filter-item><span class="file-kind">{{.Label}}</span><code>{{.Path}}</code></li>{{end}}</ul>{{else}}<p class="muted">本轮未记录到工作区文件变化。</p>{{end}}<h3>记录前已有改动：{{len .View.Changes.PreExisting}}</h3><p class="muted">开始记录前已存在，不归因于本次 AI。</p>{{if .PreExistingPreview}}<ul class="compact-list">{{range .PreExistingPreview}}<li data-filter-item><code>{{.}}</code></li>{{end}}</ul>{{end}}</section>
<section class="panel" id="evidence-boundary"><h2>证据边界</h2><div class="boundary"><strong>已记录</strong>{{if .ObservedCapabilities}}<ul>{{range .ObservedCapabilities}}<li>{{.Label}}</li>{{end}}</ul>{{else}}<p class="muted">当前没有已确认的能力范围。</p>{{end}}</div><div class="boundary missing"><strong>未监控或状态未知</strong>{{if .MissingCapabilities}}<ul>{{range .MissingCapabilities}}<li>{{.Label}}</li>{{end}}</ul>{{else}}<p>能力列表中没有未监控项。</p>{{end}}{{if .EvidenceIssues}}<ul>{{range .EvidenceIssues}}<li>{{.Label}}</li>{{end}}</ul>{{end}}</div><p class="boundary-note">未监控表示 AFR 没有这类证据，不能据此判断该行为没有发生。</p></section>
</div>
{{if .View.Risks}}<section class="panel" id="all-risks"><h2>全部风险与建议</h2>{{range .View.Risks}}<article class="risk" data-filter-item><h3 class="{{.Severity}}">{{severityLabel .Severity}} · {{.RuleID}}</h3><p>{{.Explanation}}</p>{{if .Action}}<p><strong>建议：</strong>{{.Action}}</p>{{end}}</article>{{end}}</section>{{end}}
<details class="panel" id="file-details"><summary>查看全部文件变化</summary><div class="details-body"><h2>本次文件变化</h2>{{if .Files}}<ul class="compact-list">{{range .Files}}<li data-filter-item><span class="file-kind">{{.Label}}</span><code>{{.Path}}</code></li>{{end}}</ul>{{else}}<p>无。</p>{{end}}{{if .MoreFiles}}<p class="muted">另有 {{.MoreFiles}} 条本次变化路径未在 HTML 中列出。</p>{{end}}<h2>记录前已有改动</h2><p class="muted">以下路径开始记录前已存在，不归因于本次 AI。</p>{{if .PreExisting}}<ul class="compact-list">{{range .PreExisting}}<li data-filter-item><code>{{.}}</code></li>{{end}}</ul>{{else}}<p>无。</p>{{end}}{{if .MorePreExisting}}<p class="muted">另有 {{.MorePreExisting}} 条记录前路径未在 HTML 中列出。</p>{{end}}</div></details>
<details class="panel technical" id="technical-details"><summary>技术详情</summary><div class="details-body"><p class="muted">这里保留原始枚举与有界证据，供排查和复核使用。</p><label for="filter">筛选风险、文件与事件</label><input id="filter" type="search" placeholder="输入关键词筛选本报告">
<details><summary>Session 与完整性</summary><div><dl class="metadata"><dt>Session ID</dt><dd><code>{{.View.Session.ID}}</code></dd><dt>原始状态</dt><dd><code>{{.View.Session.State}}</code></dd><dt>工作区</dt><dd><code>{{.View.Session.Workspace}}</code></dd><dt>宿主 / 捕获方式</dt><dd>{{.View.Session.Host}} / {{.View.Session.CaptureMode}}</dd><dt>开始 / 结束</dt><dd>{{.View.Session.StartedAt}} / {{.View.Session.FinishedAt}}</dd><dt>Child exit</dt><dd>{{.ChildExit}}</dd><dt>最终事件序号</dt><dd>{{.View.Session.FinalEventSeq}}</dd>{{if .View.Session.FinalHash}}<dt>最终事件哈希</dt><dd><code>{{.View.Session.FinalHash}}</code></dd>{{end}}</dl><p class="muted">本报告不冒充实时校验结果；可运行 <code>afr verify {{.View.Session.ID}}</code> 检查本地证据一致性。</p></div></details>
<details><summary>原始能力与限制</summary><div><h3>Capabilities</h3><ul>{{range .View.Session.Capabilities}}<li data-filter-item><code>{{.}}</code></li>{{else}}<li>无。</li>{{end}}</ul><h3>Limitations</h3><ul>{{range .View.Limitations}}<li data-filter-item><code>{{.}}</code></li>{{else}}<li>无。</li>{{end}}</ul></div></details>
<details><summary>事件计数</summary><div class="table-wrap"><table><thead><tr><th>事件</th><th>原始类型</th><th>数量</th></tr></thead><tbody>{{range .View.Events}}<tr data-filter-item><td>{{eventLabel .Type}}</td><td><code>{{.Type}}</code></td><td>{{.Count}}</td></tr>{{end}}</tbody></table></div></details>
<details><summary>事件时间线</summary><div class="table-wrap"><table><thead><tr><th>Seq</th><th>时间</th><th>事件</th><th>来源</th><th>摘要</th></tr></thead><tbody>{{range .View.Timeline.Items}}{{if .OmittedCount}}<tr data-filter-item><td colspan="5" class="muted">HTML 省略 {{.OmittedCount}} 条事件（seq {{.OmittedFrom}}–{{.OmittedTo}}）；完整脱敏证据仍保存在 events.jsonl。</td></tr>{{else}}<tr data-filter-item data-event-row><td>{{.Seq}}</td><td>{{.Timestamp}}</td><td>{{eventLabel .Type}}<br><code>{{.Type}}</code></td><td>{{.Source}}</td><td><code>{{.Summary}}</code>{{if .SummaryTruncated}} <span class="muted">[摘要已截断]</span>{{end}}</td></tr>{{end}}{{end}}</tbody></table></div></details>
<details><summary>进程输出</summary><div>{{range .View.Output}}<article data-filter-item><h3>{{.Stream}}</h3><p>{{.TotalBytes}} 总字节 · {{.RecordedBytes}} 已记录 · {{.Records}} 条记录 · 截断：{{.Truncated}}</p><p class="muted">session HMAC：<code>{{.Fingerprint}}</code></p>{{if .Preview}}<pre>{{.Preview}}</pre>{{end}}</article>{{else}}<p>无输出记录。</p>{{end}}<p class="muted">每个 stream 的预览最多 4 KiB；完整脱敏证据仍保存在 events.jsonl。</p></div></details>
</div></details></main><script>{{.Script}}</script></body></html>`

type htmlFile struct {
	Kind  string
	Label string
	Path  string
}

type htmlState struct {
	Label          string
	Badge          string
	Tone           string
	NeedsAttention bool
}

type htmlCapability struct {
	Label string
}

type htmlEvidenceIssue struct {
	Raw   string
	Label string
}

type htmlReportData struct {
	View                 ReportView
	Project              string
	Host                 string
	State                htmlState
	ChildExit            string
	ChangeCount          int
	AttentionCount       int
	EvidenceGapCount     int
	HasAttention         bool
	HasTopRisk           bool
	TopRisk              RiskFinding
	HighestRiskLabel     string
	ObservedCapabilities []htmlCapability
	MissingCapabilities  []htmlCapability
	EvidenceIssues       []htmlEvidenceIssue
	Files                []htmlFile
	ChangePreview        []htmlFile
	MoreFiles            int
	PreExisting          []string
	PreExistingPreview   []string
	MorePreExisting      int
	CSP                  string
	Style                template.CSS
	Script               template.JS
}

var parsedReportTemplate = template.Must(template.New("report").Funcs(template.FuncMap{
	"eventLabel":    eventLabel,
	"severityLabel": severityLabel,
}).Parse(reportTemplate))

func writeHTMLReport(sessionRoot string, view ReportView, redactor *Redactor) error {
	safeView, err := redactedReportView(view, redactor)
	if err != nil {
		return err
	}
	view = safeView
	files, moreFiles := reportFiles(view.Changes, 1000)
	remaining := 1000 - len(files)
	preExisting, morePreExisting := boundedStrings(view.Changes.PreExisting, remaining)
	observed, missing := reportCapabilities(view.Session.Capabilities)
	issues := reportEvidenceIssues(view)
	state := reportState(view.Session.State)
	attentionCount := len(view.Risks) + len(issues)
	if state.NeedsAttention {
		attentionCount++
	}
	data := htmlReportData{
		View:                 view,
		Project:              reportProject(view.Session.Workspace),
		Host:                 reportHost(view.Session),
		State:                state,
		ChildExit:            exitCodeText(view.Session.ChildExitCode),
		ChangeCount:          len(view.Changes.Added) + len(view.Changes.Modified) + len(view.Changes.Deleted) + len(view.Changes.Renamed),
		AttentionCount:       attentionCount,
		EvidenceGapCount:     len(missing) + len(issues),
		HasAttention:         attentionCount > 0,
		HasTopRisk:           len(view.Risks) > 0,
		HighestRiskLabel:     severityLabel(view.HighestRisk),
		ObservedCapabilities: observed,
		MissingCapabilities:  missing,
		EvidenceIssues:       issues,
		Files:                files,
		ChangePreview:        firstFiles(files, 4),
		MoreFiles:            moreFiles,
		PreExisting:          preExisting,
		PreExistingPreview:   firstStrings(preExisting, 3),
		MorePreExisting:      morePreExisting,
		CSP:                  "default-src 'none'; base-uri 'none'; connect-src 'none'; font-src 'none'; form-action 'none'; frame-ancestors 'none'; img-src 'none'; object-src 'none'; script-src 'sha256-" + contentHash(reportScript) + "'; style-src 'sha256-" + contentHash(reportStyle) + "'",
		Style:                template.CSS(reportStyle),
		Script:               template.JS(reportScript),
	}
	if data.HasTopRisk {
		data.TopRisk = view.Risks[0]
	}
	var output bytes.Buffer
	if err := parsedReportTemplate.Execute(&output, data); err != nil {
		return err
	}
	return atomicWriteBytes(sessionRoot, "report.html", output.Bytes())
}

func redactedReportView(view ReportView, redactor *Redactor) (ReportView, error) {
	if redactor == nil {
		return ReportView{}, fmt.Errorf("HTML report requires a redactor")
	}
	data, err := redactor.Marshal(view)
	if err != nil {
		return ReportView{}, fmt.Errorf("redact HTML report view: %w", err)
	}
	var safe ReportView
	if err := json.Unmarshal(data, &safe); err != nil {
		return ReportView{}, fmt.Errorf("decode redacted HTML report view: %w", err)
	}
	return safe, nil
}

func contentHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return base64.StdEncoding.EncodeToString(hash[:])
}

func reportState(state string) htmlState {
	switch state {
	case "idle":
		return htmlState{Label: "上一轮已记录，等待下一轮", Badge: "等待下一轮", Tone: "neutral"}
	case "starting", "running", "finalizing":
		return htmlState{Label: "正在记录当前任务", Badge: "正在记录", Tone: "active"}
	case "completed":
		return htmlState{Label: "任务已结束并保存", Badge: "已保存", Tone: "good"}
	case "failed":
		return htmlState{Label: "任务执行失败；本报告已保存，请检查失败原因", Badge: "任务失败", Tone: "warn", NeedsAttention: true}
	case "interrupted":
		return htmlState{Label: "任务被中断，请检查记录完整性", Badge: "已中断", Tone: "warn", NeedsAttention: true}
	case "incomplete":
		return htmlState{Label: "记录不完整，需要检查", Badge: "记录不完整", Tone: "warn", NeedsAttention: true}
	default:
		return htmlState{Label: "未知状态：" + state, Badge: "状态未知", Tone: "warn", NeedsAttention: true}
	}
}

func reportProject(workspace string) string {
	workspace = strings.TrimRight(strings.TrimSpace(workspace), `/\`)
	if index := strings.LastIndexAny(workspace, `/\`); index >= 0 {
		workspace = workspace[index+1:]
	}
	if workspace == "" {
		return "未命名项目"
	}
	return workspace
}

func reportHost(session SessionMetadata) string {
	if host := strings.TrimSpace(session.Host); host != "" {
		return host
	}
	if mode := strings.TrimSpace(session.CaptureMode); mode != "" {
		return mode
	}
	return "未知宿主"
}

func reportCapabilities(capabilities []string) ([]htmlCapability, []htmlCapability) {
	observed, missing := []htmlCapability{}, []htmlCapability{}
	for _, raw := range capabilities {
		name, state, ok := strings.Cut(raw, "=")
		label := capabilityLabel(name)
		if !ok {
			missing = append(missing, htmlCapability{Label: raw + "（状态未知）"})
			continue
		}
		switch state {
		case "observed":
			observed = append(observed, htmlCapability{Label: label})
		case "not_observable":
			missing = append(missing, htmlCapability{Label: label})
		default:
			missing = append(missing, htmlCapability{Label: label + "（状态未知）"})
		}
	}
	return observed, missing
}

func capabilityLabel(name string) string {
	switch name {
	case "native_tool_events":
		return "宿主上报的本地工具调用"
	case "workspace_git":
		return "Git 工作区变化"
	case "workspace_scan":
		return "项目文件扫描"
	case "local_policy":
		return "本地规则检查"
	case "process":
		return "进程级活动"
	case "network_monitor":
		return "实际网络流量"
	case "os_file_monitor":
		return "操作系统级文件活动"
	default:
		return name
	}
}

func reportEvidenceIssues(view ReportView) []htmlEvidenceIssue {
	issues := []htmlEvidenceIssue{}
	seen := map[string]bool{}
	add := func(raw, label string) {
		if raw == "" || seen[raw] {
			return
		}
		seen[raw] = true
		issues = append(issues, htmlEvidenceIssue{Raw: raw, Label: label})
	}
	for _, limitation := range view.Limitations {
		if strings.HasSuffix(limitation, "=not_observable") {
			continue
		}
		add(limitation, limitationLabel(limitation))
	}
	if view.Changes.Partial {
		add("workspace=partial", "项目文件扫描不完整")
	}
	if view.Changes.Patch.Truncated {
		add("workspace_patch=truncated", "工作区补丁已截断")
	}
	if view.Changes.Patch.Error != "" {
		add("workspace_patch_error="+view.Changes.Patch.Error, "工作区补丁不可完整生成")
	}
	if len(view.Changes.Patch.Omitted) > 0 {
		add(fmt.Sprintf("workspace_patch_omitted=%d", len(view.Changes.Patch.Omitted)), fmt.Sprintf("工作区补丁省略了 %d 个文件", len(view.Changes.Patch.Omitted)))
	}
	for _, stream := range view.Output {
		if stream.Truncated {
			add("output="+stream.Stream+"_truncated", stream.Stream+" 输出仅保留了一部分")
		}
	}
	return issues
}

func limitationLabel(raw string) string {
	switch raw {
	case "workspace=partial":
		return "项目文件扫描不完整"
	case "workspace_patch=truncated":
		return "工作区补丁已截断"
	case "native_tool_outcomes=partial":
		return "部分工具缺少结束事件，无法确认其执行结果"
	}
	lower := strings.ToLower(raw)
	if strings.Contains(lower, "binary") {
		return "部分二进制内容未进入补丁（" + raw + "）"
	}
	if strings.Contains(lower, "omitted") {
		return "部分证据被省略（" + raw + "）"
	}
	return "证据受限（" + raw + "）"
}

func severityLabel(severity string) string {
	switch severity {
	case "critical":
		return "严重"
	case "high":
		return "高风险"
	case "medium":
		return "中风险"
	case "low":
		return "低风险"
	case "none", "":
		return "无"
	default:
		return severity
	}
}

func eventLabel(eventType string) string {
	switch eventType {
	case "host_session_started", "session_started":
		return "会话开始"
	case "prompt_submitted":
		return "用户提交提示"
	case "tool_started":
		return "工具开始"
	case "tool_finished":
		return "工具结束"
	case "turn_stopped":
		return "本轮结束"
	case "process_started":
		return "进程开始"
	case "process_output":
		return "进程输出"
	case "process_finished", "session_finished":
		return "任务结束"
	case "workspace_before":
		return "工作区记录前快照"
	case "workspace_after":
		return "工作区记录后快照"
	default:
		return eventType
	}
}

func reportFiles(changes WorkspaceDelta, limit int) ([]htmlFile, int) {
	files := make([]htmlFile, 0, limit)
	appendPaths := func(kind, label string, paths []string) {
		for _, path := range paths {
			if len(files) == limit {
				return
			}
			files = append(files, htmlFile{Kind: kind, Label: label, Path: path})
		}
	}
	appendPaths("added", "新增", changes.Added)
	appendPaths("modified", "修改", changes.Modified)
	appendPaths("deleted", "删除", changes.Deleted)
	for _, rename := range changes.Renamed {
		if len(files) == limit {
			break
		}
		files = append(files, htmlFile{Kind: "renamed", Label: "重命名", Path: rename.From + " → " + rename.To})
	}
	total := len(changes.Added) + len(changes.Modified) + len(changes.Deleted) + len(changes.Renamed)
	return files, total - len(files)
}

func boundedStrings(values []string, limit int) ([]string, int) {
	if limit < 0 {
		limit = 0
	}
	if len(values) <= limit {
		return values, 0
	}
	return values[:limit], len(values) - limit
}

func firstFiles(values []htmlFile, limit int) []htmlFile {
	if len(values) <= limit {
		return values
	}
	return values[:limit]
}

func firstStrings(values []string, limit int) []string {
	if len(values) <= limit {
		return values
	}
	return values[:limit]
}
