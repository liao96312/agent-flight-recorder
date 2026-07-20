package afr

import (
	"strings"
	"testing"
)

func TestRedactorBuiltInDetectors(t *testing.T) {
	redactor := testRedactor(t)
	secrets := []string{
		"sk-" + strings.Repeat("a", 24),
		"ghp_" + strings.Repeat("b", 30),
		"AKIA" + "ABCDEFGHIJKLMNOP",
		"Bearer " + strings.Repeat("c", 24),
		"postgres://user:pass@example.invalid/db",
		"password=hunter2",
		strings.Join([]string{"-----BEGIN " + "PRIVATE KEY-----", "abc", "-----END " + "PRIVATE KEY-----"}, "\n"),
	}
	for _, secret := range secrets {
		redacted := redactor.Text(secret)
		if strings.Contains(redacted, secret) || !strings.Contains(redacted, "[REDACTED:") {
			t.Fatalf("secret not redacted: %q => %q", secret, redacted)
		}
	}
}

func TestRedactorLeavesBenignTextAlone(t *testing.T) {
	redactor := testRedactor(t)
	for _, text := range []string{"token bucket algorithm", "password policy", "https://example.com/docs", "secret management overview"} {
		if redacted := redactor.Text(text); redacted != text {
			t.Fatalf("false positive: %q => %q", text, redacted)
		}
	}
}

func TestRecursiveMarshalRedactsPathsErrorsAndKeys(t *testing.T) {
	redactor := testRedactor(t)
	data, err := redactor.Marshal(map[string]any{
		"error":  "failed at password=hunter2",
		"path":   `C:\\token=abcdefghijklmnop\\file.txt`,
		"nested": []string{"cookie=abcdefghijklmnop"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"hunter2", "abcdefghijklmnop"} {
		if strings.Contains(string(data), secret) {
			t.Fatalf("secret %q remained in %s", secret, data)
		}
	}
}

func TestArgvRedactionPreservesChildBoundary(t *testing.T) {
	redactor := testRedactor(t)
	original := []string{"agent", "--token", "small", "--password=hunter2", "plain"}
	redacted := redactor.Argv(original)
	if original[2] != "small" || redacted[2] == "small" || strings.Contains(redacted[3], "hunter2") || redacted[4] != "plain" {
		t.Fatalf("original=%v redacted=%v", original, redacted)
	}
}

func TestRedactionPlaceholderIsSessionScoped(t *testing.T) {
	first := testRedactor(t)
	second := testRedactor(t)
	secret := "token=repeatable-secret"
	firstValue := first.Text(secret)
	if firstValue != first.Text(secret) || firstValue == second.Text(secret) {
		t.Fatal("redaction correlation must be stable only within one session")
	}
}

func TestRecordAccumulatorHandlesChunksANSIAndPrivateKey(t *testing.T) {
	redactor := testRedactor(t)
	accumulator := NewRecordAccumulator(redactor, 1024)
	records := accumulator.Feed([]byte("prefix sk-abcdefgh"))
	if len(records) != 0 {
		t.Fatal("partial chunk emitted early")
	}
	records = append(records, accumulator.Feed([]byte("ijklmnop \x1b[31mred\x1b[0m\n"+"-----BEGIN "+"PRIVATE KEY-----\nabc\n"))...)
	records = append(records, accumulator.Feed([]byte("-----END "+"PRIVATE KEY-----\r\n"))...)
	if len(records) != 2 || strings.Contains(records[0].Text, "sk-") || strings.Contains(records[0].Text, "\x1b") || strings.Contains(records[1].Text, "PRIVATE KEY") {
		t.Fatalf("records=%+v", records)
	}
}

func TestRecordAccumulatorFailsClosed(t *testing.T) {
	redactor := testRedactor(t)
	long := NewRecordAccumulator(redactor, 4)
	records := long.Feed([]byte("secret\n"))
	if len(records) != 1 || records[0].OmittedReason != "record_too_long" || records[0].Fingerprint == "" || records[0].Text != "" {
		t.Fatalf("long=%+v", records)
	}
	binary := NewRecordAccumulator(redactor, 32)
	records = binary.Feed([]byte{'a', 0, 'b', '\n'})
	if len(records) != 1 || records[0].OmittedReason != "binary_or_unknown_encoding" || records[0].Text != "" {
		t.Fatalf("binary=%+v", records)
	}
}

func testRedactor(t *testing.T) *Redactor {
	t.Helper()
	fingerprinter, err := NewFingerprinter()
	if err != nil {
		t.Fatal(err)
	}
	redactor, err := NewRedactor(fingerprinter)
	if err != nil {
		t.Fatal(err)
	}
	return redactor
}
