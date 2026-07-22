package afr

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

const maxWorkspaceConfigBytes = 1024 * 1024

type workspaceConfig struct {
	RedactionRules []redactionRuleConfig `json:"redaction_rules"`
}

type redactionRuleConfig struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Pattern string `json:"pattern"`
}

var redactionRuleName = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

func loadWorkspaceRedactionRules(root string) ([]secretDetector, error) {
	file, err := os.Open(filepath.Join(root, ".afr.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read .afr.json: %w", err)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxWorkspaceConfigBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read .afr.json: %w", err)
	}
	if len(data) > maxWorkspaceConfigBytes {
		return nil, fmt.Errorf(".afr.json exceeds %d bytes", maxWorkspaceConfigBytes)
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var config workspaceConfig
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("decode .afr.json: %w", err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return nil, fmt.Errorf("decode .afr.json: %w", err)
	}
	if config.RedactionRules == nil {
		return nil, errors.New(".afr.json requires redaction_rules")
	}

	seen := make(map[string]bool, len(secretDetectors)+len(config.RedactionRules))
	for _, detector := range secretDetectors {
		seen[detector.name] = true
	}
	detectors := make([]secretDetector, 0, len(config.RedactionRules))
	for index, rule := range config.RedactionRules {
		if !redactionRuleName.MatchString(rule.Name) {
			return nil, fmt.Errorf("redaction_rules[%d].name must match %s", index, redactionRuleName)
		}
		if seen[rule.Name] {
			return nil, fmt.Errorf("duplicate redaction rule name %q", rule.Name)
		}
		seen[rule.Name] = true
		if rule.Type != "regex" {
			return nil, fmt.Errorf("redaction_rules[%d].type must be %q", index, "regex")
		}
		if rule.Pattern == "" {
			return nil, fmt.Errorf("redaction_rules[%d].pattern must not be empty", index)
		}
		pattern, err := regexp.Compile(rule.Pattern)
		if err != nil {
			return nil, fmt.Errorf("compile redaction rule %q: %w", rule.Name, err)
		}
		if pattern.MatchString("") {
			return nil, fmt.Errorf("redaction rule %q must not match empty text", rule.Name)
		}
		detectors = append(detectors, secretDetector{name: rule.Name, pattern: pattern})
	}
	return detectors, nil
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("multiple JSON values")
	}
	return err
}
