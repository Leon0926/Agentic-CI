package findings

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fixedReport is deliberately hand-built rather than run through a real
// detector: this test locks in the encoder's shape, not detector
// behavior, which already has its own tests. Meta.GeneratedAt is fixed —
// without that this test goes red on every run regardless of any real
// change.
func fixedReport() *Report {
	return &Report{
		SchemaVersion: 1,
		Meta: Meta{
			ReviewdVersion: "test",
			RepoSHA:        "abc123",
			Ref:            "HEAD",
			Model:          "claude-test",
			Threshold:      0.5,
			GeneratedAt:    time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
		},
		Detectors: []DetectorStatus{
			{Name: "secrets", Status: "ran", DurationMS: 5},
			{Name: "secrets-llm", PromptVersion: "v1", Status: "ran", DurationMS: 1300},
		},
		Findings: []Finding{
			{
				File:        filepath.FromSlash("/repo/internal/foo.go"),
				Line:        12,
				Detector:    "secrets",
				RuleID:      "aws-access-key-id",
				Severity:    SeverityHigh,
				Confidence:  0.9,
				Explanation: "Possible AWS access key ID committed in the diff.",
			},
			{
				// deliberately low confidence: proves WriteJSON keeps it
				// while a text-view test elsewhere would drop it.
				File:        filepath.FromSlash("/repo/internal/bar.go"),
				Line:        3,
				Detector:    "secrets-llm",
				RuleID:      "hardcoded-token",
				Severity:    SeverityMedium,
				Confidence:  0.3,
				Explanation: "Looks like a token, but low confidence.",
			},
		},
	}
}

func TestWriteJSON_Golden(t *testing.T) {
	report := fixedReport()
	require.NoError(t, report.Normalize(filepath.FromSlash("/repo")))

	var buf bytes.Buffer
	require.NoError(t, WriteJSON(&buf, report))

	golden := filepath.Join("testdata", "golden.json")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		require.NoError(t, os.WriteFile(golden, buf.Bytes(), 0o644))
	}

	want, err := os.ReadFile(golden)
	require.NoError(t, err, "run with UPDATE_GOLDEN=1 to (re)generate testdata/golden.json")
	require.Equal(t, string(want), buf.String())
}

func TestWriteJSON_NeverFiltersByConfidence(t *testing.T) {
	report := fixedReport()
	require.NoError(t, report.Normalize(filepath.FromSlash("/repo")))

	var buf bytes.Buffer
	require.NoError(t, WriteJSON(&buf, report))

	require.Contains(t, buf.String(), "hardcoded-token", "low-confidence finding must still be present in JSON")
}

func TestWriteText_DropsBelowThreshold(t *testing.T) {
	report := fixedReport()
	require.NoError(t, report.Normalize(filepath.FromSlash("/repo")))

	var buf bytes.Buffer
	require.NoError(t, WriteText(&buf, report, 0.5))

	require.Contains(t, buf.String(), "aws-access-key-id", "0.9-confidence finding should be shown at threshold 0.5")
	require.NotContains(t, buf.String(), "hardcoded-token", "0.3-confidence finding should be dropped at threshold 0.5")
}
