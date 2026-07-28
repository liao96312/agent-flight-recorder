package afr

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	defaultScanFiles = 100_000
	defaultScanBytes = int64(512 * 1024 * 1024)
	defaultScanTime  = 5 * time.Second
	maxGitOutput     = int64(64 * 1024 * 1024)
)

type Fingerprinter struct {
	key [32]byte
}

type ScanLimits struct {
	MaxFiles int
	MaxBytes int64
	MaxTime  time.Duration
}

type FileEvidence struct {
	Path           string `json:"path"`
	Type           string `json:"type"`
	Size           int64  `json:"size,omitempty"`
	ModifiedAt     string `json:"modified_at,omitempty"`
	Fingerprint    string `json:"fingerprint,omitempty"`
	LinkTarget     string `json:"link_target,omitempty"`
	ResolvedTarget string `json:"resolved_target,omitempty"`
	TargetOutside  bool   `json:"target_outside,omitempty"`
	OmittedReason  string `json:"omitted_reason,omitempty"`
}

type GitEntry struct {
	Kind         string `json:"kind"`
	XY           string `json:"xy,omitempty"`
	Submodule    string `json:"submodule,omitempty"`
	Path         string `json:"path"`
	OriginalPath string `json:"original_path,omitempty"`
}

type GitEvidence struct {
	Available bool       `json:"available"`
	Version   string     `json:"version,omitempty"`
	RepoRoot  string     `json:"repo_root,omitempty"`
	Branch    string     `json:"branch,omitempty"`
	Head      string     `json:"head,omitempty"`
	Entries   []GitEntry `json:"entries,omitempty"`
	Error     string     `json:"error,omitempty"`
}

type WorkspaceSnapshot struct {
	FormatVersion int            `json:"format_version"`
	CapturedAt    string         `json:"captured_at"`
	Root          string         `json:"root"`
	Mode          string         `json:"mode"`
	Files         []FileEvidence `json:"files"`
	Git           GitEvidence    `json:"git"`
	Partial       bool           `json:"partial"`
	Omissions     []string       `json:"omissions,omitempty"`
	ScannedFiles  int            `json:"scanned_files"`
	ReadBytes     int64          `json:"read_bytes"`
	ElapsedMS     int64          `json:"elapsed_ms"`
}

type RenameEvidence struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type WorkspaceDelta struct {
	FormatVersion   int                   `json:"format_version"`
	Added           []string              `json:"added"`
	Modified        []string              `json:"modified"`
	Deleted         []string              `json:"deleted"`
	Renamed         []RenameEvidence      `json:"renamed"`
	PreExisting     []string              `json:"pre_existing"`
	Partial         bool                  `json:"partial"`
	OmissionReasons []string              `json:"omission_reasons,omitempty"`
	Patch           WorkspacePatchSummary `json:"patch"`
}

func WriteWorkspaceArtifact(sessionRoot, name string, value any, redactor *Redactor) error {
	directory, err := safeJoin(sessionRoot, "snapshots")
	if err != nil {
		return err
	}
	if err := os.Mkdir(directory, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		return fmt.Errorf("create snapshot directory: %w", err)
	}
	return atomicWriteJSON(sessionRoot, filepath.Join("snapshots", name), value, redactor)
}

func WorkspaceCapabilities(snapshot WorkspaceSnapshot) []string {
	gitCapability := "workspace_git=not_observable"
	if snapshot.Git.Available && snapshot.Git.Error == "" {
		gitCapability = "workspace_git=observed"
	}
	return []string{
		"process=observed",
		gitCapability,
		"workspace_scan=observed",
		"native_tool_events=not_observable",
		"local_policy=observed",
		"os_file_monitor=not_observable",
		"network_monitor=not_observable",
	}
}

func NewFingerprinter() (*Fingerprinter, error) {
	fingerprinter := &Fingerprinter{}
	if _, err := rand.Read(fingerprinter.key[:]); err != nil {
		return nil, fmt.Errorf("create session fingerprint key: %w", err)
	}
	return fingerprinter, nil
}

func NewFingerprinterFromKey(key []byte) (*Fingerprinter, error) {
	if len(key) != sha256.Size {
		return nil, fmt.Errorf("fingerprinter key must be %d bytes", sha256.Size)
	}
	fingerprinter := &Fingerprinter{}
	copy(fingerprinter.key[:], key)
	return fingerprinter, nil
}

func (fingerprinter *Fingerprinter) Bytes(data []byte) string {
	hash := hmac.New(sha256.New, fingerprinter.key[:])
	_, _ = hash.Write(data)
	return hex.EncodeToString(hash.Sum(nil))
}

func (fingerprinter *Fingerprinter) NewHash() hash.Hash {
	return hmac.New(sha256.New, fingerprinter.key[:])
}

func CollectWorkspace(root, excludedRoot string, fingerprinter *Fingerprinter, limits ScanLimits) WorkspaceSnapshot {
	ignore, err := loadAFRIgnore(root)
	if err != nil {
		snapshot := collectWorkspace(root, excludedRoot, fingerprinter, limits, afrIgnore{})
		snapshot.Partial = true
		snapshot.Omissions = append(snapshot.Omissions, "afrignore_invalid")
		sort.Strings(snapshot.Omissions)
		snapshot.Omissions = compactStrings(snapshot.Omissions)
		return snapshot
	}
	return collectWorkspace(root, excludedRoot, fingerprinter, limits, ignore)
}

func collectWorkspace(root, excludedRoot string, fingerprinter *Fingerprinter, limits ScanLimits, ignore afrIgnore) WorkspaceSnapshot {
	started := time.Now()
	if limits.MaxFiles <= 0 {
		limits.MaxFiles = defaultScanFiles
	}
	if limits.MaxBytes <= 0 {
		limits.MaxBytes = defaultScanBytes
	}
	if limits.MaxTime <= 0 {
		limits.MaxTime = defaultScanTime
	}
	snapshot := WorkspaceSnapshot{
		FormatVersion: 1,
		CapturedAt:    time.Now().UTC().Format(time.RFC3339Nano),
		Root:          root,
		Mode:          "scan",
		Files:         []FileEvidence{},
		Git:           collectGit(root),
	}
	if snapshot.Git.Available {
		snapshot.Mode = "git"
	}

	deadline := started.Add(limits.MaxTime)
	errStop := errors.New("scan budget reached")
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			snapshot.Partial = true
			snapshot.Omissions = append(snapshot.Omissions, "walk_error")
			return nil
		}
		if samePath(path, excludedRoot) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil || !filepath.IsLocal(relative) {
			snapshot.Partial = true
			snapshot.Omissions = append(snapshot.Omissions, "path_outside_root")
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if relative == "." {
			return nil
		}
		if ignore.matches(relative) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if time.Now().After(deadline) || snapshot.ScannedFiles >= limits.MaxFiles {
			snapshot.Partial = true
			snapshot.Omissions = append(snapshot.Omissions, "scan_budget")
			return errStop
		}

		record := FileEvidence{Path: filepath.ToSlash(relative)}
		info, infoErr := entry.Info()
		if infoErr != nil {
			record.Type = "unknown"
			record.OmittedReason = "stat_error"
			snapshot.Partial = true
			snapshot.Files = append(snapshot.Files, record)
			return nil
		}
		snapshot.ScannedFiles++
		record.ModifiedAt = info.ModTime().UTC().Format(time.RFC3339Nano)
		if isLinkLike(info) {
			record.Type = "symlink"
			target, linkErr := os.Readlink(path)
			if linkErr != nil {
				record.OmittedReason = "readlink_error"
				snapshot.Partial = true
			} else {
				record.LinkTarget = target
				if !filepath.IsAbs(target) {
					target = filepath.Join(filepath.Dir(path), target)
				}
				if resolvedTarget, resolveErr := filepath.EvalSymlinks(target); resolveErr == nil {
					target = resolvedTarget
					record.ResolvedTarget = resolvedTarget
				} else {
					record.OmittedReason = "link_target_unresolved"
					snapshot.Partial = true
				}
				record.TargetOutside = !pathWithin(root, target)
			}
			snapshot.Files = append(snapshot.Files, record)
			return nil
		}
		if !info.Mode().IsRegular() {
			record.Type = "other"
			snapshot.Files = append(snapshot.Files, record)
			return nil
		}
		record.Type = "file"
		record.Size = info.Size()
		if info.Size() > limits.MaxBytes-snapshot.ReadBytes {
			record.OmittedReason = "byte_budget"
			snapshot.Files = append(snapshot.Files, record)
			snapshot.Partial = true
			snapshot.Omissions = append(snapshot.Omissions, "byte_budget")
			return errStop
		}
		file, openErr := os.Open(path)
		if openErr != nil {
			record.OmittedReason = "open_error"
			snapshot.Partial = true
			snapshot.Files = append(snapshot.Files, record)
			return nil
		}
		hash := hmac.New(sha256.New, fingerprinter.key[:])
		read, copyErr := io.Copy(hash, io.LimitReader(file, limits.MaxBytes-snapshot.ReadBytes+1))
		closeErr := file.Close()
		snapshot.ReadBytes += read
		if copyErr != nil || closeErr != nil || read != info.Size() {
			record.OmittedReason = "read_changed_or_failed"
			snapshot.Partial = true
		} else {
			record.Fingerprint = hex.EncodeToString(hash.Sum(nil))
		}
		snapshot.Files = append(snapshot.Files, record)
		return nil
	})
	if err != nil && !errors.Is(err, errStop) {
		snapshot.Partial = true
		snapshot.Omissions = append(snapshot.Omissions, "walk_failed")
	}
	sort.Slice(snapshot.Files, func(i, j int) bool { return snapshot.Files[i].Path < snapshot.Files[j].Path })
	sort.Strings(snapshot.Omissions)
	snapshot.Omissions = compactStrings(snapshot.Omissions)
	snapshot.ElapsedMS = time.Since(started).Milliseconds()
	return snapshot
}

func CompareWorkspace(before, after WorkspaceSnapshot) WorkspaceDelta {
	delta := WorkspaceDelta{FormatVersion: 1, Added: []string{}, Modified: []string{}, Deleted: []string{}, Renamed: []RenameEvidence{}, PreExisting: []string{}}
	beforeFiles := make(map[string]FileEvidence, len(before.Files))
	afterFiles := make(map[string]FileEvidence, len(after.Files))
	for _, file := range before.Files {
		beforeFiles[file.Path] = file
	}
	for _, file := range after.Files {
		afterFiles[file.Path] = file
	}
	deletedRecords := map[string]FileEvidence{}
	addedRecords := map[string]FileEvidence{}
	for path, file := range beforeFiles {
		afterFile, exists := afterFiles[path]
		if !exists {
			deletedRecords[path] = file
			continue
		}
		if file.Type != afterFile.Type || file.Size != afterFile.Size || file.ModifiedAt != afterFile.ModifiedAt || file.Fingerprint != afterFile.Fingerprint || file.LinkTarget != afterFile.LinkTarget || file.ResolvedTarget != afterFile.ResolvedTarget {
			delta.Modified = append(delta.Modified, path)
		}
	}
	for path, file := range afterFiles {
		if _, exists := beforeFiles[path]; !exists {
			addedRecords[path] = file
		}
	}
	deletedPaths := sortedKeys(deletedRecords)
	addedPaths := sortedKeys(addedRecords)
	usedAdded := map[string]bool{}
	// ponytail: identical-content renames pair by sorted path; prefer Git rename metadata if ambiguity matters in real sessions.
	for _, deletedPath := range deletedPaths {
		deleted := deletedRecords[deletedPath]
		matched := ""
		if deleted.Fingerprint != "" {
			for _, addedPath := range addedPaths {
				added := addedRecords[addedPath]
				if !usedAdded[addedPath] && deleted.Type == added.Type && deleted.Fingerprint == added.Fingerprint {
					matched = addedPath
					break
				}
			}
		}
		if matched != "" {
			usedAdded[matched] = true
			delta.Renamed = append(delta.Renamed, RenameEvidence{From: deletedPath, To: matched})
		} else {
			delta.Deleted = append(delta.Deleted, deletedPath)
		}
	}
	for _, addedPath := range addedPaths {
		if !usedAdded[addedPath] {
			delta.Added = append(delta.Added, addedPath)
		}
	}
	for _, entry := range before.Git.Entries {
		delta.PreExisting = append(delta.PreExisting, entry.Path)
	}
	sort.Strings(delta.Added)
	sort.Strings(delta.Modified)
	sort.Strings(delta.Deleted)
	sort.Strings(delta.PreExisting)
	delta.PreExisting = compactStrings(delta.PreExisting)
	delta.Partial = before.Partial || after.Partial
	delta.OmissionReasons = append(delta.OmissionReasons, before.Omissions...)
	delta.OmissionReasons = append(delta.OmissionReasons, after.Omissions...)
	sort.Strings(delta.OmissionReasons)
	delta.OmissionReasons = compactStrings(delta.OmissionReasons)
	return delta
}

func collectGit(workspace string) GitEvidence {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return GitEvidence{Error: "git_not_found"}
	}
	evidence := GitEvidence{}
	if output, err := runGit(gitPath, workspace, "version"); err == nil {
		evidence.Version = strings.TrimSpace(string(output))
	}
	repoRoot, err := runGit(gitPath, workspace, "rev-parse", "--show-toplevel")
	if err != nil {
		var commandError *gitCommandError
		if errors.As(err, &commandError) && strings.Contains(commandError.stderr, "not a git repository") {
			evidence.Error = "not_a_git_repository"
		} else {
			evidence.Error = "git_probe_failed"
		}
		return evidence
	}
	evidence.Available = true
	evidence.RepoRoot = strings.TrimSpace(string(repoRoot))
	if branch, branchErr := runGit(gitPath, workspace, "symbolic-ref", "--short", "-q", "HEAD"); branchErr == nil {
		evidence.Branch = strings.TrimSpace(string(branch))
	}
	if head, headErr := runGit(gitPath, workspace, "rev-parse", "--verify", "HEAD"); headErr == nil {
		evidence.Head = strings.TrimSpace(string(head))
	}
	status, err := runGit(gitPath, workspace, "status", "--porcelain=v2", "-z", "--branch", "--untracked-files=all", "--", ".")
	if err != nil {
		evidence.Error = "git_status_failed"
		return evidence
	}
	entries, err := parsePorcelainV2(status)
	if err != nil {
		evidence.Error = "git_status_parse_failed"
		return evidence
	}
	evidence.Entries = entries
	return evidence
}

func runGit(gitPath, workspace string, args ...string) ([]byte, error) {
	commandArgs := append([]string{"--no-optional-locks", "-c", "color.ui=false", "-c", "core.quotepath=false", "-C", workspace}, args...)
	command := exec.Command(gitPath, commandArgs...)
	command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_PAGER=cat", "GIT_ASKPASS=", "LC_ALL=C", "LANG=C")
	stdout := &limitedBuffer{limit: maxGitOutput}
	stderr := &limitedBuffer{limit: 64 * 1024}
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		return nil, &gitCommandError{cause: err, stderr: stderr.String()}
	}
	if stdout.exceeded {
		return nil, errors.New("git output exceeded limit")
	}
	return stdout.Bytes(), nil
}

type gitCommandError struct {
	cause  error
	stderr string
}

func (err *gitCommandError) Error() string { return "git command failed" }
func (err *gitCommandError) Unwrap() error { return err.cause }

func parsePorcelainV2(output []byte) ([]GitEntry, error) {
	records := bytes.Split(output, []byte{0})
	entries := []GitEntry{}
	for index := 0; index < len(records); index++ {
		record := string(records[index])
		if record == "" || strings.HasPrefix(record, "# ") {
			continue
		}
		switch record[0] {
		case '1':
			fields := strings.SplitN(record, " ", 9)
			if len(fields) != 9 {
				return nil, errors.New("invalid ordinary git status record")
			}
			entries = append(entries, GitEntry{Kind: "ordinary", XY: fields[1], Submodule: fields[2], Path: fields[8]})
		case '2':
			fields := strings.SplitN(record, " ", 10)
			if len(fields) != 10 || index+1 >= len(records) {
				return nil, errors.New("invalid rename git status record")
			}
			index++
			entries = append(entries, GitEntry{Kind: "rename", XY: fields[1], Submodule: fields[2], Path: fields[9], OriginalPath: string(records[index])})
		case 'u':
			fields := strings.SplitN(record, " ", 11)
			if len(fields) != 11 {
				return nil, errors.New("invalid unmerged git status record")
			}
			entries = append(entries, GitEntry{Kind: "unmerged", XY: fields[1], Submodule: fields[2], Path: fields[10]})
		case '?':
			if len(record) < 3 {
				return nil, errors.New("invalid untracked git status record")
			}
			entries = append(entries, GitEntry{Kind: "untracked", Path: record[2:]})
		case '!':
			continue
		default:
			return nil, errors.New("unknown git status record")
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Path == entries[j].Path {
			return entries[i].Kind < entries[j].Kind
		}
		return entries[i].Path < entries[j].Path
	})
	return entries, nil
}

type limitedBuffer struct {
	bytes.Buffer
	limit    int64
	exceeded bool
}

func (buffer *limitedBuffer) Write(data []byte) (int, error) {
	original := len(data)
	remaining := buffer.limit - int64(buffer.Len())
	if remaining <= 0 {
		buffer.exceeded = true
		return original, nil
	}
	if int64(len(data)) > remaining {
		data = data[:remaining]
		buffer.exceeded = true
	}
	_, _ = buffer.Buffer.Write(data)
	return original, nil
}

func pathWithin(root, target string) bool {
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	relative, err := filepath.Rel(root, absTarget)
	return err == nil && filepath.IsLocal(relative)
}

func samePath(left, right string) bool {
	if left == "" || right == "" {
		return false
	}
	leftAbs, leftErr := filepath.Abs(left)
	rightAbs, rightErr := filepath.Abs(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	leftInfo, leftStatErr := os.Stat(leftAbs)
	rightInfo, rightStatErr := os.Stat(rightAbs)
	if leftStatErr == nil && rightStatErr == nil {
		return os.SameFile(leftInfo, rightInfo)
	}
	return filepath.Clean(leftAbs) == filepath.Clean(rightAbs)
}

func compactStrings(values []string) []string {
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

func sortedKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
