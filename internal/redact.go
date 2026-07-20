package afr

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

const defaultRecordLimit = 1024 * 1024

type secretDetector struct {
	name    string
	pattern *regexp.Regexp
}

var secretDetectors = []secretDetector{
	{name: "private_key", pattern: regexp.MustCompile(`(?s)-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----.*?-----END [A-Z0-9 ]*PRIVATE KEY-----`)},
	{name: "connection_string", pattern: regexp.MustCompile(`(?i)\b(?:postgres(?:ql)?|mysql|mongodb(?:\+srv)?|redis|amqp)://[^\s"'<>]+`)},
	{name: "url_credentials", pattern: regexp.MustCompile(`(?i)\bhttps?://[^/\s:@]+:[^@\s]+@[^\s"'<>]+`)},
	{name: "bearer", pattern: regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]{12,}`)},
	{name: "github_token", pattern: regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9_]{20,}\b`)},
	{name: "openai_key", pattern: regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{16,}\b`)},
	{name: "aws_key", pattern: regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)},
	{name: "sensitive_value", pattern: regexp.MustCompile(`(?i)\b(?:api[_-]?key|access[_-]?token|auth(?:orization)?|cookie|credential|password|passwd|pwd|secret|token)\s*[:=]\s*[^\s,;]{4,}`)},
}

var ansiPattern = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)

type Redactor struct {
	fingerprinter *Fingerprinter
}

func NewRedactor(fingerprinter *Fingerprinter) (*Redactor, error) {
	if fingerprinter == nil {
		return nil, errors.New("redactor requires a session fingerprinter")
	}
	return &Redactor{fingerprinter: fingerprinter}, nil
}

func (redactor *Redactor) Text(text string) string {
	text, _ = redactor.TextWithKinds(text)
	return text
}

func (redactor *Redactor) TextWithKinds(text string) (string, []string) {
	kinds := []string{}
	for _, detector := range secretDetectors {
		text = detector.pattern.ReplaceAllStringFunc(text, func(secret string) string {
			kinds = append(kinds, detector.name)
			return redactor.placeholder(detector.name, secret)
		})
	}
	sort.Strings(kinds)
	return text, compactStrings(kinds)
}

func (redactor *Redactor) Argv(argv []string) []string {
	result, _ := redactor.ArgvWithKinds(argv)
	return result
}

func (redactor *Redactor) ArgvWithKinds(argv []string) ([]string, []string) {
	result := make([]string, len(argv))
	kinds := []string{}
	sensitiveNext := false
	for index, argument := range argv {
		if sensitiveNext {
			result[index] = redactor.placeholder("argv", argument)
			kinds = append(kinds, "argv")
			sensitiveNext = false
			continue
		}
		if name, value, found := strings.Cut(argument, "="); found && sensitiveFlag(name) {
			result[index] = name + "=" + redactor.placeholder("argv", value)
			kinds = append(kinds, "argv")
			continue
		}
		var detected []string
		result[index], detected = redactor.TextWithKinds(argument)
		kinds = append(kinds, detected...)
		sensitiveNext = (strings.HasPrefix(argument, "-") || strings.HasPrefix(argument, "/")) && sensitiveFlag(argument)
	}
	sort.Strings(kinds)
	return result, compactStrings(kinds)
}

func (redactor *Redactor) EnvironmentNames(environment []string) []string {
	names := make([]string, 0, len(environment))
	for _, entry := range environment {
		name, _, _ := strings.Cut(entry, "=")
		names = append(names, redactor.Text(name))
	}
	sort.Strings(names)
	return compactStrings(names)
}

func (redactor *Redactor) Marshal(value any) ([]byte, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var generic any
	if err := decoder.Decode(&generic); err != nil {
		return nil, err
	}
	return json.Marshal(redactor.sanitize(generic))
}

func (redactor *Redactor) MarshalIndent(value any) ([]byte, error) {
	safe, err := redactor.Marshal(value)
	if err != nil {
		return nil, err
	}
	buffer := &bytes.Buffer{}
	if err := json.Indent(buffer, safe, "", "  "); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (redactor *Redactor) sanitize(value any) any {
	switch typed := value.(type) {
	case string:
		return redactor.Text(typed)
	case []any:
		for index := range typed {
			typed[index] = redactor.sanitize(typed[index])
		}
		return typed
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, item := range typed {
			result[redactor.Text(key)] = redactor.sanitize(item)
		}
		return result
	default:
		return value
	}
}

func (redactor *Redactor) placeholder(kind, secret string) string {
	fingerprint := redactor.fingerprinter.Bytes([]byte(secret))
	return fmt.Sprintf("[REDACTED:%s:%s]", kind, fingerprint[:12])
}

func sensitiveFlag(flag string) bool {
	flag = strings.ToLower(strings.TrimLeft(flag, "-\\/"))
	flag = strings.ReplaceAll(flag, "-", "_")
	for _, marker := range []string{"api_key", "apikey", "auth", "cookie", "credential", "password", "passwd", "secret", "token"} {
		if strings.Contains(flag, marker) {
			return true
		}
	}
	return false
}

type RedactedRecord struct {
	Text          string
	Bytes         int
	OmittedReason string
	Fingerprint   string
	Redactions    []string
}

type RecordAccumulator struct {
	redactor      *Redactor
	limit         int
	buffer        []byte
	bytes         int
	omitted       bool
	privateKey    bool
	privateKeyEnd bool
	markerWindow  []byte
	fingerprint   hash.Hash
}

func NewRecordAccumulator(redactor *Redactor, limit int) *RecordAccumulator {
	if limit <= 0 {
		limit = defaultRecordLimit
	}
	accumulator := &RecordAccumulator{redactor: redactor, limit: limit}
	accumulator.reset()
	return accumulator
}

func (accumulator *RecordAccumulator) Feed(data []byte) []RedactedRecord {
	records := []RedactedRecord{}
	for len(data) > 0 {
		newline := bytes.IndexByte(data, '\n')
		if newline < 0 {
			accumulator.consume(data)
			break
		}
		segment := data[:newline+1]
		accumulator.consume(segment)
		data = data[newline+1:]
		if accumulator.privateKey && !accumulator.privateKeyEnd {
			continue
		}
		records = append(records, accumulator.finish(""))
	}
	return records
}

func (accumulator *RecordAccumulator) Close() []RedactedRecord {
	if accumulator.bytes == 0 {
		return nil
	}
	reason := ""
	if accumulator.privateKey && !accumulator.privateKeyEnd {
		reason = "unterminated_private_key"
	}
	return []RedactedRecord{accumulator.finish(reason)}
}

func (accumulator *RecordAccumulator) consume(data []byte) {
	accumulator.bytes += len(data)
	_, _ = accumulator.fingerprint.Write(data)
	accumulator.markerWindow = append(accumulator.markerWindow, data...)
	if len(accumulator.markerWindow) > 256 {
		accumulator.markerWindow = accumulator.markerWindow[len(accumulator.markerWindow)-256:]
	}
	marker := string(accumulator.markerWindow)
	if strings.Contains(marker, "-----BEGIN ") && strings.Contains(marker, "PRIVATE KEY-----") {
		accumulator.privateKey = true
	}
	if accumulator.privateKey && strings.Contains(marker, "-----END ") && strings.Contains(marker, "PRIVATE KEY-----") {
		accumulator.privateKeyEnd = true
	}
	if accumulator.omitted {
		return
	}
	if len(accumulator.buffer)+len(data) > accumulator.limit {
		accumulator.omitted = true
		accumulator.buffer = nil
		return
	}
	accumulator.buffer = append(accumulator.buffer, data...)
}

func (accumulator *RecordAccumulator) finish(reason string) RedactedRecord {
	record := RedactedRecord{Bytes: accumulator.bytes}
	if reason == "" && accumulator.omitted {
		reason = "record_too_long"
	}
	trimmed := bytes.TrimSuffix(accumulator.buffer, []byte{'\n'})
	trimmed = bytes.TrimSuffix(trimmed, []byte{'\r'})
	if reason == "" && (bytes.IndexByte(trimmed, 0) >= 0 || !utf8.Valid(trimmed)) {
		reason = "binary_or_unknown_encoding"
	}
	if reason != "" {
		record.OmittedReason = reason
		record.Fingerprint = fmt.Sprintf("%x", accumulator.fingerprint.Sum(nil))
		if accumulator.privateKey {
			record.Redactions = []string{"private_key"}
		}
	} else {
		record.Text, record.Redactions = accumulator.redactor.TextWithKinds(ansiPattern.ReplaceAllString(string(trimmed), ""))
	}
	accumulator.reset()
	return record
}

func (accumulator *RecordAccumulator) reset() {
	accumulator.buffer = accumulator.buffer[:0]
	accumulator.bytes = 0
	accumulator.omitted = false
	accumulator.privateKey = false
	accumulator.privateKeyEnd = false
	accumulator.markerWindow = accumulator.markerWindow[:0]
	accumulator.fingerprint = accumulator.redactor.fingerprinter.NewHash()
}
