package afr

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"
)

var sessionIDPattern = regexp.MustCompile(`^[0-9]{8}T[0-9]{6}\.[0-9]{9}Z-[0-9a-f]{24}$`)

type CleanOptions struct {
	OlderThan time.Duration
	MaxBytes  int64
	Now       time.Time
}

type CleanTarget struct {
	ID        string `json:"id"`
	Path      string `json:"path"`
	Bytes     int64  `json:"bytes"`
	Timestamp string `json:"timestamp"`
}

type CleanPlan struct {
	Targets    []CleanTarget `json:"targets"`
	TotalBytes int64         `json:"total_bytes"`
	root       string
}

type cleanCandidate struct {
	CleanTarget
	time   time.Time
	active bool
}

func PlanClean(sessionsRoot string, options CleanOptions) (CleanPlan, error) {
	if (options.OlderThan > 0) == (options.MaxBytes > 0) {
		return CleanPlan{}, errors.New("clean requires exactly one positive retention condition")
	}
	if options.Now.IsZero() {
		options.Now = time.Now()
	}
	entries, err := os.ReadDir(sessionsRoot)
	if errors.Is(err, os.ErrNotExist) {
		return CleanPlan{Targets: []CleanTarget{}, root: sessionsRoot}, nil
	}
	if err != nil {
		return CleanPlan{}, fmt.Errorf("read sessions: %w", err)
	}
	candidates := []cleanCandidate{}
	for _, entry := range entries {
		info, infoErr := entry.Info()
		if infoErr != nil {
			return CleanPlan{}, fmt.Errorf("inspect session entry %s: %w", entry.Name(), infoErr)
		}
		if isLinkLike(info) {
			return CleanPlan{}, fmt.Errorf("refuse linked session entry %s", entry.Name())
		}
		if !info.IsDir() {
			continue
		}
		if !sessionIDPattern.MatchString(entry.Name()) {
			return CleanPlan{}, fmt.Errorf("refuse forged session directory %s", entry.Name())
		}
		root, err := safeJoin(sessionsRoot, entry.Name())
		if err != nil || !isRealDirectory(root) {
			return CleanPlan{}, fmt.Errorf("refuse invalid session directory %s", entry.Name())
		}
		candidate, err := inspectCleanCandidate(root, entry.Name())
		if err != nil {
			return CleanPlan{}, err
		}
		candidates = append(candidates, candidate)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].time.Equal(candidates[j].time) {
			return candidates[i].ID < candidates[j].ID
		}
		return candidates[i].time.Before(candidates[j].time)
	})
	plan := CleanPlan{Targets: []CleanTarget{}, root: sessionsRoot}
	if options.OlderThan > 0 {
		cutoff := options.Now.Add(-options.OlderThan)
		for _, candidate := range candidates {
			if candidate.time.After(cutoff) {
				continue
			}
			if candidate.active {
				return CleanPlan{}, fmt.Errorf("refuse active session %s", candidate.ID)
			}
			plan.Targets = append(plan.Targets, candidate.CleanTarget)
			plan.TotalBytes += candidate.Bytes
		}
		return plan, nil
	}
	var currentBytes int64
	for _, candidate := range candidates {
		currentBytes += candidate.Bytes
	}
	for _, candidate := range candidates {
		if currentBytes <= options.MaxBytes {
			break
		}
		if candidate.active {
			continue
		}
		plan.Targets = append(plan.Targets, candidate.CleanTarget)
		plan.TotalBytes += candidate.Bytes
		currentBytes -= candidate.Bytes
	}
	if currentBytes > options.MaxBytes {
		return CleanPlan{}, errors.New("capacity target cannot be reached without an active session")
	}
	return plan, nil
}

func ExecuteClean(plan CleanPlan) ([]CleanTarget, error) {
	if len(plan.Targets) == 0 {
		return []CleanTarget{}, nil
	}
	root, err := filepath.Abs(plan.root)
	if err != nil || !isRealDirectory(root) {
		return nil, errors.New("clean plan has an invalid sessions root")
	}
	for _, target := range plan.Targets {
		expected, joinErr := safeJoin(root, target.ID)
		if joinErr != nil || expected != target.Path || filepath.Dir(expected) != root || !sessionIDPattern.MatchString(target.ID) {
			return nil, fmt.Errorf("refuse changed clean target %s", target.ID)
		}
		candidate, inspectErr := inspectCleanCandidate(expected, target.ID)
		if inspectErr != nil || candidate.active || candidate.Bytes != target.Bytes || candidate.Timestamp != target.Timestamp {
			return nil, fmt.Errorf("refuse changed or active clean target %s", target.ID)
		}
	}
	deleted := make([]CleanTarget, 0, len(plan.Targets))
	for _, target := range plan.Targets {
		quarantine, joinErr := safeJoin(root, ".deleting-"+target.ID)
		if joinErr != nil {
			return deleted, joinErr
		}
		if _, statErr := os.Lstat(quarantine); !errors.Is(statErr, os.ErrNotExist) {
			return deleted, fmt.Errorf("refuse occupied clean quarantine for %s", target.ID)
		}
		if err := os.Rename(target.Path, quarantine); err != nil {
			return deleted, fmt.Errorf("isolate clean target %s: %w", target.ID, err)
		}
		if err := os.RemoveAll(quarantine); err != nil {
			return deleted, fmt.Errorf("delete clean target %s: %w", target.ID, err)
		}
		deleted = append(deleted, target)
	}
	return deleted, nil
}

func inspectCleanCandidate(root, directoryID string) (cleanCandidate, error) {
	metadataPath, err := safeJoin(root, "session.json")
	if err != nil || !isBoundedRegularFile(metadataPath, maxSessionBytes) {
		return cleanCandidate{}, fmt.Errorf("refuse session %s with invalid metadata", directoryID)
	}
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return cleanCandidate{}, fmt.Errorf("read session %s metadata: %w", directoryID, err)
	}
	var metadata SessionMetadata
	if json.Unmarshal(data, &metadata) != nil || metadata.FormatVersion != 1 || metadata.ID != directoryID {
		return cleanCandidate{}, fmt.Errorf("refuse forged session metadata %s", directoryID)
	}
	timestamp := metadata.FinishedAt
	if timestamp == "" {
		timestamp = metadata.StartedAt
	}
	parsed, err := time.Parse(time.RFC3339Nano, timestamp)
	if err != nil {
		return cleanCandidate{}, fmt.Errorf("refuse session %s with invalid timestamp", directoryID)
	}
	active := metadata.RecorderPID > 0 && processAlive(metadata.RecorderPID)
	if metadata.State != "completed" && metadata.State != "failed" && metadata.RecorderPID == 0 {
		return cleanCandidate{}, fmt.Errorf("refuse session %s with unverifiable active state", directoryID)
	}
	size, err := cleanSessionSize(root)
	if err != nil {
		return cleanCandidate{}, fmt.Errorf("refuse session %s: %w", directoryID, err)
	}
	return cleanCandidate{CleanTarget: CleanTarget{ID: directoryID, Path: root, Bytes: size, Timestamp: timestamp}, time: parsed, active: active}, nil
}

func cleanSessionSize(root string) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if isLinkLike(info) {
			return errors.New("linked artifact")
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return errors.New("non-regular artifact")
		}
		total += info.Size()
		return nil
	})
	return total, err
}
