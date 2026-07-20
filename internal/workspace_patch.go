package afr

import (
	"bytes"
	"errors"
	"fmt"
	"hash"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const defaultDiffLimit = int64(50 * 1024 * 1024)

type PatchOmission struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type WorkspacePatchSummary struct {
	Path        string          `json:"path"`
	Included    []string        `json:"included"`
	Omitted     []PatchOmission `json:"omitted,omitempty"`
	TotalBytes  int64           `json:"total_bytes"`
	Fingerprint string          `json:"fingerprint"`
	Truncated   bool            `json:"truncated"`
	Error       string          `json:"error,omitempty"`
}

func GenerateWorkspacePatch(workspace, sessionRoot string, before, after WorkspaceSnapshot, delta WorkspaceDelta, fingerprinter *Fingerprinter, redactor *Redactor, limit int64) (WorkspacePatchSummary, error) {
	summary := WorkspacePatchSummary{Path: "diffs/workspace.patch", Included: []string{}}
	if err := os.Mkdir(filepath.Join(sessionRoot, "diffs"), 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		return summary, fmt.Errorf("create diff directory: %w", err)
	}
	if limit <= 0 {
		limit = defaultDiffLimit
	}
	if !before.Git.Available || !after.Git.Available || before.Git.Error != "" || after.Git.Error != "" || before.Git.Head == "" {
		summary.Error = "git_diff_not_observable"
		return summary, writePatchResult(sessionRoot, summary, "# AFR: Git diff not observable; see workspace-delta.json for file summaries.\n", redactor)
	}

	preExisting := stringSet(delta.PreExisting)
	for _, rename := range delta.Renamed {
		if preExisting[rename.From] {
			preExisting[rename.To] = true
		}
	}
	paths := deltaPaths(delta)
	untracked := gitPaths(after.Git.Entries, func(entry GitEntry) bool { return entry.Kind == "untracked" })
	submodules := gitPaths(append(append([]GitEntry{}, before.Git.Entries...), after.Git.Entries...), func(entry GitEntry) bool {
		return entry.Submodule != "" && entry.Submodule != "N..."
	})
	tracked := []string{}
	newFiles := []string{}
	for _, path := range paths {
		if coveredBy(path, preExisting) {
			summary.Omitted = append(summary.Omitted, PatchOmission{Path: path, Reason: "pre_existing"})
			continue
		}
		if root := coveringRoot(path, submodules); root != "" {
			summary.Omitted = append(summary.Omitted, PatchOmission{Path: root, Reason: "submodule"})
			continue
		}
		if isLFS(workspace, path) {
			summary.Omitted = append(summary.Omitted, PatchOmission{Path: path, Reason: "lfs"})
			continue
		}
		if untracked[path] {
			if fileLooksBinary(filepath.Join(workspace, filepath.FromSlash(path))) {
				summary.Omitted = append(summary.Omitted, PatchOmission{Path: path, Reason: "binary"})
			} else {
				newFiles = append(newFiles, path)
			}
			continue
		}
		tracked = append(tracked, path)
	}
	sort.Strings(tracked)
	sort.Strings(newFiles)
	summary.Omitted = compactOmissions(summary.Omitted)

	capture := newDiffCapture(fingerprinter, limit)
	if len(tracked) > 0 {
		// ponytail: one pathspec command has the Windows command-line ceiling; batch only if the large-repo benchmark reaches it.
		args := append([]string{"diff", "--no-ext-diff", "--no-textconv", "--src-prefix=a/", "--dst-prefix=b/", before.Git.Head, "--"}, tracked...)
		if err := runGitStream(workspace, capture, false, args...); err != nil {
			summary.Error = "git_diff_failed"
		}
	}
	for _, path := range newFiles {
		if summary.Error != "" {
			break
		}
		if err := runGitStream(workspace, capture, true, "diff", "--no-index", "--no-ext-diff", "--no-textconv", "--src-prefix=a/", "--dst-prefix=b/", "--", "/dev/null", path); err != nil {
			summary.Error = "git_diff_failed"
			break
		}
	}
	summary.Included = append(tracked, newFiles...)
	sort.Strings(summary.Included)
	summary.TotalBytes = capture.total
	summary.Fingerprint = fmt.Sprintf("%x", capture.fingerprint.Sum(nil))
	summary.Truncated = capture.truncated

	body := capture.buffer.String()
	if summary.Error != "" {
		body = "# AFR: Git diff failed; see workspace-delta.json for file summaries.\n"
	} else if summary.Truncated {
		body = "# AFR: patch omitted because the 50 MB evidence limit was exceeded.\n"
	} else if body == "" {
		body = "# AFR: no attributable text changes.\n"
	}
	for _, omitted := range summary.Omitted {
		body += fmt.Sprintf("# AFR omitted %s: %s\n", omitted.Reason, omitted.Path)
	}
	return summary, writePatchResult(sessionRoot, summary, body, redactor)
}

func writePatchResult(sessionRoot string, summary WorkspacePatchSummary, body string, redactor *Redactor) error {
	return atomicWriteRedactedText(sessionRoot, summary.Path, body, redactor)
}

func deltaPaths(delta WorkspaceDelta) []string {
	set := stringSet(append(append(append([]string{}, delta.Added...), delta.Modified...), delta.Deleted...))
	for _, rename := range delta.Renamed {
		set[rename.From] = true
		set[rename.To] = true
	}
	return sortedKeys(set)
}

func stringSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}

func gitPaths(entries []GitEntry, match func(GitEntry) bool) map[string]bool {
	paths := map[string]bool{}
	for _, entry := range entries {
		if match(entry) {
			paths[entry.Path] = true
		}
	}
	return paths
}

func coveredBy(path string, roots map[string]bool) bool { return coveringRoot(path, roots) != "" }

func coveringRoot(path string, roots map[string]bool) string {
	for root := range roots {
		if path == root || strings.HasPrefix(path, strings.TrimSuffix(root, "/")+"/") {
			return root
		}
	}
	return ""
}

func isLFS(workspace, path string) bool {
	git, err := exec.LookPath("git")
	if err != nil {
		return false
	}
	output, err := runGit(git, workspace, "check-attr", "filter", "--", path)
	return err == nil && strings.HasSuffix(strings.TrimSpace(string(output)), ": lfs")
}

func fileLooksBinary(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	data := make([]byte, 8*1024)
	n, _ := file.Read(data)
	data = data[:n]
	return bytes.IndexByte(data, 0) >= 0 || !utf8.Valid(data)
}

type diffCapture struct {
	buffer      bytes.Buffer
	limit       int64
	total       int64
	truncated   bool
	fingerprint hash.Hash
}

func newDiffCapture(fingerprinter *Fingerprinter, limit int64) *diffCapture {
	return &diffCapture{limit: limit, fingerprint: fingerprinter.NewHash()}
}

func (capture *diffCapture) Write(data []byte) (int, error) {
	capture.total += int64(len(data))
	_, _ = capture.fingerprint.Write(data)
	if capture.total > capture.limit {
		capture.truncated = true
		capture.buffer.Reset()
		return len(data), nil
	}
	if !capture.truncated {
		_, _ = capture.buffer.Write(data)
	}
	return len(data), nil
}

func runGitStream(workspace string, stdout *diffCapture, acceptDifference bool, args ...string) error {
	git, err := exec.LookPath("git")
	if err != nil {
		return err
	}
	commandArgs := append([]string{"--no-optional-locks", "-c", "color.ui=false", "-c", "core.quotepath=false", "-C", workspace}, args...)
	command := exec.Command(git, commandArgs...)
	command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_PAGER=cat", "GIT_ASKPASS=", "LC_ALL=C", "LANG=C")
	command.Stdout = stdout
	command.Stderr = &limitedBuffer{limit: 64 * 1024}
	err = command.Run()
	if acceptDifference {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) && exitError.ExitCode() == 1 {
			return nil
		}
	}
	return err
}

func compactOmissions(values []PatchOmission) []PatchOmission {
	sort.Slice(values, func(i, j int) bool {
		if values[i].Path == values[j].Path {
			return values[i].Reason < values[j].Reason
		}
		return values[i].Path < values[j].Path
	})
	result := values[:0]
	for _, value := range values {
		if len(result) == 0 || result[len(result)-1] != value {
			result = append(result, value)
		}
	}
	return result
}
