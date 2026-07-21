package afr

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
)

type ManifestEntry struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type Manifest struct {
	FormatVersion int             `json:"format_version"`
	SessionID     string          `json:"session_id"`
	Evidence      []ManifestEntry `json:"evidence"`
	Derived       []ManifestEntry `json:"derived"`
}

func WriteManifest(sessionRoot, sessionID string, evidencePaths, derivedPaths []string, redactor *Redactor) error {
	evidence, err := manifestEntries(sessionRoot, evidencePaths)
	if err != nil {
		return err
	}
	derived, err := manifestEntries(sessionRoot, derivedPaths)
	if err != nil {
		return err
	}
	manifest := Manifest{FormatVersion: ManifestFormatVersion, SessionID: sessionID, Evidence: evidence, Derived: derived}
	return atomicWriteJSON(sessionRoot, "manifest.json", manifest, redactor)
}

func manifestEntries(sessionRoot string, paths []string) ([]ManifestEntry, error) {
	paths = append([]string(nil), paths...)
	sort.Strings(paths)
	entries := make([]ManifestEntry, 0, len(paths))
	for index, relative := range paths {
		if relative == "manifest.json" {
			return nil, fmt.Errorf("manifest cannot include itself")
		}
		if index > 0 && relative == paths[index-1] {
			return nil, fmt.Errorf("manifest artifact %s is duplicated", relative)
		}
		path, err := safeJoin(sessionRoot, relative)
		if err != nil {
			return nil, fmt.Errorf("manifest path %q: %w", relative, err)
		}
		info, err := os.Lstat(path)
		if err != nil {
			return nil, fmt.Errorf("inspect manifest artifact %s: %w", relative, err)
		}
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("manifest artifact %s is not a regular file", relative)
		}
		file, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open manifest artifact %s: %w", relative, err)
		}
		openInfo, statErr := file.Stat()
		if statErr != nil || !openInfo.Mode().IsRegular() {
			_ = file.Close()
			if statErr != nil {
				return nil, fmt.Errorf("inspect manifest artifact %s: %w", relative, statErr)
			}
			return nil, fmt.Errorf("manifest artifact %s is not a regular file", relative)
		}
		hash := sha256.New()
		written, copyErr := io.Copy(hash, file)
		closeErr := file.Close()
		if copyErr != nil || closeErr != nil {
			return nil, fmt.Errorf("hash manifest artifact %s: %w", relative, errors.Join(copyErr, closeErr))
		}
		if written != info.Size() {
			return nil, fmt.Errorf("hash manifest artifact %s: artifact changed while hashing", relative)
		}
		entries = append(entries, ManifestEntry{Path: relative, Size: written, SHA256: hex.EncodeToString(hash.Sum(nil))})
	}
	return entries, nil
}
