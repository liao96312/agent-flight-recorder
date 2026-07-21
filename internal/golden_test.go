package afr

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFormatV1GoldenVectors(t *testing.T) {
	redactor, err := NewRedactor(&Fingerprinter{})
	if err != nil {
		t.Fatal(err)
	}

	if got, want := redactor.Text("password=hunter2"), "[REDACTED:sensitive_value:0788fb16ceca]"; got != want {
		t.Fatalf("redaction golden changed:\n%q", got)
	}

	finding, found := CommandRisk([]string{"git", "reset", "--hard"}, 7)
	if !found {
		t.Fatal("risk golden input was not detected")
	}
	risk, err := json.Marshal(finding)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(risk), `{"rule_id":"command.destructive","rule_version":1,"severity":"high","evidence_seq":[7],"observed":["visible argv matched destructive action: git reset --hard"],"inferred":"the requested command can delete or irreversibly replace data","explanation":"AFR matched only the command argv it directly observed; child-internal commands remain not observable.","action":"Confirm the target and ensure a recoverable backup exists."}`; got != want {
		t.Fatalf("risk golden changed:\n%s", got)
	}

	root := t.TempDir()
	for name, content := range map[string]string{
		"events.jsonl": "event\n",
		"session.json": "session\n",
		"report.html":  "derived\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := WriteManifest(root, "golden-session", []string{"session.json", "events.jsonl"}, []string{"report.html"}, redactor); err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	const manifestGolden = `{
  "derived": [
    {
      "path": "report.html",
      "sha256": "ae514718289bb8d8313cf7271bb500fe7f9d6d5f49daac0cf3e9dc1927826b38",
      "size": 8
    }
  ],
  "evidence": [
    {
      "path": "events.jsonl",
      "sha256": "d8073d788ee641f2f54333c3246b08951f721c3f8090cdcb1f0fa9e80eaef504",
      "size": 6
    },
    {
      "path": "session.json",
      "sha256": "e9d6e33fc2fea3757c6a1e121276a412ca43028a7a1efc9b6a68b1182bd55c8e",
      "size": 8
    }
  ],
  "format_version": 1,
  "session_id": "golden-session"
}
`
	if got, want := string(manifest), manifestGolden; got != want {
		t.Fatalf("manifest golden changed:\n%s", got)
	}
}
