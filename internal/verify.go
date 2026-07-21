package afr

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	maxManifestBytes = 16 * 1024 * 1024
	maxSessionBytes  = 1024 * 1024
)

var requiredEvidencePaths = []string{
	"diffs/workspace.patch",
	"events.jsonl",
	"session.json",
	"snapshots/workspace-after.json",
	"snapshots/workspace-before.json",
	"snapshots/workspace-delta.json",
}

type ArtifactState struct {
	Exists bool   `json:"exists"`
	Size   int64  `json:"size,omitempty"`
	SHA256 string `json:"sha256,omitempty"`
}

type VerificationIssue struct {
	Kind     string         `json:"kind"`
	Code     string         `json:"code"`
	Seq      uint64         `json:"seq,omitempty"`
	Path     string         `json:"path,omitempty"`
	Message  string         `json:"message"`
	Expected string         `json:"expected,omitempty"`
	Actual   string         `json:"actual,omitempty"`
	Want     *ArtifactState `json:"expected_artifact,omitempty"`
	Got      *ArtifactState `json:"actual_artifact,omitempty"`
}

type VerifyResult struct {
	SessionID       string             `json:"session_id,omitempty"`
	Valid           bool               `json:"valid"`
	Events          uint64             `json:"events"`
	EvidenceChecked int                `json:"evidence_checked"`
	DerivedChecked  int                `json:"derived_checked"`
	Issue           *VerificationIssue `json:"issue,omitempty"`
}

func VerifySession(sessionRoot string) VerifyResult {
	result := VerifyResult{}
	manifest, issue := readManifestForVerify(sessionRoot)
	if issue != nil {
		result.Issue = issue
		return result
	}
	result.SessionID = manifest.SessionID
	if issue = validateManifest(manifest); issue != nil {
		result.Issue = issue
		return result
	}

	inspection, eventIssue, err := inspectEventStream(sessionRoot)
	result.Events = inspection.ValidEvents
	if err != nil {
		code := "artifact_unreadable"
		if errors.Is(err, os.ErrNotExist) {
			code = "artifact_missing"
		}
		result.Issue = &VerificationIssue{Kind: "artifact", Code: code, Path: "events.jsonl", Message: err.Error()}
		return result
	}
	if eventIssue != nil {
		result.Issue = eventIssue
		return result
	}
	metadata, issue := readSessionForVerify(sessionRoot)
	if issue != nil {
		result.Issue = issue
		return result
	}
	if result.SessionID != metadata.ID {
		result.Issue = &VerificationIssue{Kind: "session", Code: "session_id_mismatch", Path: "session.json", Message: "session ID differs from manifest", Expected: result.SessionID, Actual: metadata.ID}
		return result
	}
	if metadata.FinalEventSeq != inspection.ValidEvents {
		result.Issue = &VerificationIssue{Kind: "event", Code: "event_count", Seq: inspection.ValidEvents + 1, Path: "events.jsonl", Message: "event count differs from final session metadata", Expected: fmt.Sprint(metadata.FinalEventSeq), Actual: fmt.Sprint(inspection.ValidEvents)}
		return result
	}
	if metadata.FinalHash != inspection.FinalHash {
		result.Issue = &VerificationIssue{Kind: "session", Code: "final_hash_mismatch", Path: "session.json", Message: "final event hash differs from event chain", Expected: metadata.FinalHash, Actual: inspection.FinalHash}
		return result
	}
	if inspection.LastEventType != "session_finished" {
		result.Issue = &VerificationIssue{Kind: "event", Code: "event_final_type", Seq: inspection.ValidEvents, Path: "events.jsonl", Message: "final event is not session_finished", Expected: "session_finished", Actual: inspection.LastEventType}
		return result
	}

	for _, entry := range manifest.Evidence {
		result.EvidenceChecked++
		if issue := verifyArtifact(sessionRoot, entry); issue != nil {
			result.Issue = issue
			return result
		}
	}
	for _, entry := range manifest.Derived {
		result.DerivedChecked++
		if issue := verifyArtifact(sessionRoot, entry); issue != nil {
			result.Issue = issue
			return result
		}
	}
	result.Valid = true
	return result
}

func ResolveSessionRoot(sessionsRoot, selector string) (string, error) {
	if selector == "" || !filepath.IsLocal(selector) || filepath.Base(selector) != selector {
		return "", errors.New("session selector is required")
	}
	if selector == "latest" {
		entries, err := os.ReadDir(sessionsRoot)
		if err != nil {
			return "", fmt.Errorf("read sessions: %w", err)
		}
		names := []string{}
		for _, entry := range entries {
			if entry.IsDir() && filepath.IsLocal(entry.Name()) && isRealDirectory(filepath.Join(sessionsRoot, entry.Name())) {
				names = append(names, entry.Name())
			}
		}
		if len(names) == 0 {
			return "", errors.New("no sessions found")
		}
		sort.Sort(sort.Reverse(sort.StringSlice(names)))
		selector = names[0]
	}
	root, err := safeJoin(sessionsRoot, selector)
	if err != nil {
		return "", err
	}
	if isRealDirectory(root) {
		return root, nil
	}
	entries, readErr := os.ReadDir(sessionsRoot)
	if readErr != nil {
		return "", fmt.Errorf("read sessions: %w", readErr)
	}
	matches := []string{}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), selector) && filepath.IsLocal(entry.Name()) && isRealDirectory(filepath.Join(sessionsRoot, entry.Name())) {
			matches = append(matches, entry.Name())
		}
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("find session %s: %w", selector, os.ErrNotExist)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("session prefix %q is ambiguous", selector)
	}
	return safeJoin(sessionsRoot, matches[0])
}

func isRealDirectory(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0
}

func readManifestForVerify(sessionRoot string) (Manifest, *VerificationIssue) {
	path, err := safeJoin(sessionRoot, "manifest.json")
	if err != nil {
		return Manifest{}, &VerificationIssue{Kind: "manifest", Code: "manifest_path_invalid", Path: "manifest.json", Message: err.Error()}
	}
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Manifest{}, &VerificationIssue{Kind: "manifest", Code: "manifest_missing", Path: "manifest.json", Message: "manifest is missing"}
		}
		return Manifest{}, &VerificationIssue{Kind: "manifest", Code: "manifest_unreadable", Path: "manifest.json", Message: "manifest cannot be inspected"}
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() > maxManifestBytes {
		return Manifest{}, &VerificationIssue{Kind: "manifest", Code: "manifest_invalid_file", Path: "manifest.json", Message: "manifest must be a bounded regular file"}
	}
	file, err := os.Open(path)
	if err != nil {
		return Manifest{}, &VerificationIssue{Kind: "manifest", Code: "manifest_unreadable", Path: "manifest.json", Message: "manifest cannot be read"}
	}
	defer file.Close()
	var manifest Manifest
	decoder := json.NewDecoder(io.LimitReader(file, maxManifestBytes+1))
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, &VerificationIssue{Kind: "manifest", Code: "manifest_invalid_json", Path: "manifest.json", Message: "manifest JSON is invalid"}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return Manifest{}, &VerificationIssue{Kind: "manifest", Code: "manifest_trailing_data", Path: "manifest.json", Message: "manifest contains trailing data"}
	}
	return manifest, nil
}

func validateManifest(manifest Manifest) *VerificationIssue {
	if manifest.FormatVersion != ManifestFormatVersion {
		return &VerificationIssue{Kind: "manifest", Code: "manifest_version", Path: "manifest.json", Message: "unsupported manifest format", Expected: "1", Actual: fmt.Sprint(manifest.FormatVersion)}
	}
	if manifest.SessionID == "" {
		return &VerificationIssue{Kind: "manifest", Code: "manifest_session_id", Path: "manifest.json", Message: "manifest session ID is empty"}
	}
	if !entriesSorted(manifest.Evidence) || !entriesSorted(manifest.Derived) {
		return &VerificationIssue{Kind: "manifest", Code: "manifest_order", Path: "manifest.json", Message: "manifest entries are not sorted"}
	}
	seen := map[string]bool{}
	evidence := map[string]bool{}
	for _, entry := range manifest.Evidence {
		evidence[entry.Path] = true
	}
	for _, entry := range append(append([]ManifestEntry{}, manifest.Evidence...), manifest.Derived...) {
		hash, hashErr := hex.DecodeString(entry.SHA256)
		if entry.Path == "manifest.json" || !filepath.IsLocal(entry.Path) || seen[entry.Path] || entry.Size < 0 || hashErr != nil || len(hash) != sha256.Size {
			return &VerificationIssue{Kind: "manifest", Code: "manifest_entry_invalid", Path: entry.Path, Message: "manifest entry is unsafe, duplicated, or self-referential"}
		}
		seen[entry.Path] = true
	}
	for _, required := range requiredEvidencePaths {
		if !evidence[required] {
			return &VerificationIssue{Kind: "manifest", Code: "manifest_evidence_missing", Path: required, Message: "required evidence entry is missing"}
		}
	}
	return nil
}

func entriesSorted(entries []ManifestEntry) bool {
	return sort.SliceIsSorted(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
}

func readSessionForVerify(sessionRoot string) (SessionMetadata, *VerificationIssue) {
	path, err := safeJoin(sessionRoot, "session.json")
	if err != nil {
		return SessionMetadata{}, &VerificationIssue{Kind: "session", Code: "session_path_invalid", Path: "session.json", Message: err.Error()}
	}
	info, err := os.Lstat(path)
	if err != nil {
		return SessionMetadata{}, &VerificationIssue{Kind: "session", Code: "session_unreadable", Path: "session.json", Message: "session metadata cannot be read"}
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return SessionMetadata{}, &VerificationIssue{Kind: "session", Code: "session_invalid_file", Path: "session.json", Message: "session metadata must be a regular file"}
	}
	if info.Size() > maxSessionBytes {
		return SessionMetadata{}, &VerificationIssue{Kind: "session", Code: "session_too_large", Path: "session.json", Message: "session metadata exceeds size limit"}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return SessionMetadata{}, &VerificationIssue{Kind: "session", Code: "session_unreadable", Path: "session.json", Message: "session metadata cannot be read"}
	}
	var metadata SessionMetadata
	if json.Unmarshal(data, &metadata) != nil || metadata.FormatVersion != SessionFormatVersion {
		return SessionMetadata{}, &VerificationIssue{Kind: "session", Code: "session_invalid", Path: "session.json", Message: "session metadata is invalid"}
	}
	return metadata, nil
}

func verifyArtifact(sessionRoot string, entry ManifestEntry) *VerificationIssue {
	want := &ArtifactState{Exists: true, Size: entry.Size, SHA256: entry.SHA256}
	path, err := safeJoin(sessionRoot, entry.Path)
	if err != nil {
		return &VerificationIssue{Kind: "artifact", Code: "artifact_path_invalid", Path: entry.Path, Message: "artifact path is invalid", Want: want, Got: &ArtifactState{}}
	}
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &VerificationIssue{Kind: "artifact", Code: "artifact_missing", Path: entry.Path, Message: "artifact is missing", Want: want, Got: &ArtifactState{}}
		}
		return &VerificationIssue{Kind: "artifact", Code: "artifact_unreadable", Path: entry.Path, Message: "artifact cannot be inspected", Want: want, Got: &ArtifactState{}}
	}
	got := &ArtifactState{Exists: true, Size: info.Size()}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return &VerificationIssue{Kind: "artifact", Code: "artifact_not_regular", Path: entry.Path, Message: "artifact is not a regular file", Want: want, Got: got}
	}
	file, err := os.Open(path)
	if err != nil {
		return &VerificationIssue{Kind: "artifact", Code: "artifact_unreadable", Path: entry.Path, Message: "artifact cannot be read", Want: want, Got: got}
	}
	hash := sha256.New()
	written, copyErr := io.Copy(hash, file)
	closeErr := file.Close()
	if copyErr != nil || closeErr != nil {
		return &VerificationIssue{Kind: "artifact", Code: "artifact_unreadable", Path: entry.Path, Message: "artifact could not be hashed", Want: want, Got: got}
	}
	got.Size = written
	got.SHA256 = hex.EncodeToString(hash.Sum(nil))
	if got.Size != want.Size || got.SHA256 != want.SHA256 {
		return &VerificationIssue{Kind: "artifact", Code: "artifact_mismatch", Path: entry.Path, Message: "artifact size or hash differs", Want: want, Got: got}
	}
	return nil
}
