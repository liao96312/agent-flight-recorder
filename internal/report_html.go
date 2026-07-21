package afr

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"html/template"
)

const reportStyle = `body{font:15px system-ui,sans-serif;line-height:1.5;max-width:1100px;margin:auto;padding:2rem;color:#172033;background:#f6f8fb}header,section{background:white;border:1px solid #dbe2ea;border-radius:12px;padding:1rem 1.25rem;margin-bottom:1rem}h1,h2{margin-top:0}input{box-sizing:border-box;width:100%;padding:.7rem;border:1px solid #9aa8b8;border-radius:8px}table{width:100%;border-collapse:collapse}th,td{text-align:left;padding:.45rem;border-bottom:1px solid #e6ebf0}code,pre{background:#eef2f6;border-radius:5px;padding:.15rem .3rem}pre{padding:.75rem;overflow:auto;white-space:pre-wrap}.high,.critical{color:#a40000}.medium{color:#8a5200}.muted{color:#526273}[hidden]{display:none}`

const reportScript = `(()=>{const q=document.getElementById('filter');const apply=()=>{const value=q.value.toLowerCase();document.querySelectorAll('[data-filter-item]').forEach(item=>item.hidden=!item.textContent.toLowerCase().includes(value))};q.addEventListener('input',apply)})();`

const reportTemplate = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<meta http-equiv="Content-Security-Policy" content="{{.CSP}}"><title>AFR {{.View.Session.ID}}</title><style>{{.Style}}</style></head>
<body><header><h1>Agent Flight Report</h1><p><strong>{{.View.Session.State}}</strong> · session <code>{{.View.Session.ID}}</code> · {{.View.Session.FinalEventSeq}} events · highest risk <strong>{{.View.HighestRisk}}</strong></p>
<label for="filter">Filter events, risks, and files</label><input id="filter" type="search" placeholder="Type to filter this report"></header>
<main><section><h2>Capabilities</h2><ul>{{range .View.Session.Capabilities}}<li><code>{{.}}</code></li>{{end}}</ul></section>
<section><h2>Event counts</h2><table><thead><tr><th>Type</th><th>Count</th></tr></thead><tbody>{{range .View.Events}}<tr data-filter-item><td>{{.Type}}</td><td>{{.Count}}</td></tr>{{end}}</tbody></table></section>
<section><h2>Output</h2>{{range .View.Output}}<article data-filter-item><h3>{{.Stream}}</h3><p>{{.TotalBytes}} total bytes · {{.RecordedBytes}} recorded · {{.Records}} records · truncated: {{.Truncated}}</p><p class="muted">session HMAC: <code>{{.Fingerprint}}</code></p>{{if .Preview}}<pre>{{.Preview}}</pre>{{end}}</article>{{end}}<p class="muted">Preview is limited to 4 KiB per stream; full redacted evidence remains in events.jsonl.</p></section>
<section><h2>Risks</h2>{{if .View.Risks}}{{range .View.Risks}}<article data-filter-item><h3 class="{{.Severity}}">{{.Severity}} · {{.RuleID}}</h3><p>{{.Explanation}}</p><p><strong>Action:</strong> {{.Action}}</p></article>{{end}}{{else}}<p>No deterministic risk findings.</p>{{end}}</section>
<section><h2>Workspace files</h2><p>{{len .View.Changes.Added}} added · {{len .View.Changes.Modified}} modified · {{len .View.Changes.Deleted}} deleted · {{len .View.Changes.Renamed}} renamed</p><ul>{{range .Files}}<li data-filter-item><strong>{{.Kind}}</strong> <code>{{.Path}}</code></li>{{end}}</ul>{{if .MoreFiles}}<p class="muted">{{.MoreFiles}} more paths omitted from HTML.</p>{{end}}</section>
{{if .View.Limitations}}<section><h2>Limitations</h2><ul>{{range .View.Limitations}}<li><code>{{.}}</code></li>{{end}}</ul></section>{{end}}</main><script>{{.Script}}</script></body></html>`

type htmlFile struct {
	Kind string
	Path string
}

type htmlReportData struct {
	View      ReportView
	Files     []htmlFile
	MoreFiles int
	CSP       string
	Style     template.CSS
	Script    template.JS
}

var parsedReportTemplate = template.Must(template.New("report").Parse(reportTemplate))

func writeHTMLReport(sessionRoot string, view ReportView, redactor *Redactor) error {
	files, more := reportFiles(view.Changes, 1000)
	data := htmlReportData{
		View:      view,
		Files:     files,
		MoreFiles: more,
		CSP:       "default-src 'none'; base-uri 'none'; connect-src 'none'; font-src 'none'; form-action 'none'; frame-ancestors 'none'; img-src 'none'; object-src 'none'; script-src 'sha256-" + contentHash(reportScript) + "'; style-src 'sha256-" + contentHash(reportStyle) + "'",
		Style:     template.CSS(reportStyle),
		Script:    template.JS(reportScript),
	}
	var output bytes.Buffer
	if err := parsedReportTemplate.Execute(&output, data); err != nil {
		return err
	}
	return atomicWriteRedactedText(sessionRoot, "report.html", output.String(), redactor)
}

func contentHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return base64.StdEncoding.EncodeToString(hash[:])
}

func reportFiles(changes WorkspaceDelta, limit int) ([]htmlFile, int) {
	files := make([]htmlFile, 0, limit)
	appendPaths := func(kind string, paths []string) {
		for _, path := range paths {
			if len(files) == limit {
				return
			}
			files = append(files, htmlFile{Kind: kind, Path: path})
		}
	}
	appendPaths("added", changes.Added)
	appendPaths("modified", changes.Modified)
	appendPaths("deleted", changes.Deleted)
	for _, rename := range changes.Renamed {
		if len(files) == limit {
			break
		}
		files = append(files, htmlFile{Kind: "renamed", Path: rename.From + " → " + rename.To})
	}
	total := len(changes.Added) + len(changes.Modified) + len(changes.Deleted) + len(changes.Renamed)
	return files, total - len(files)
}
