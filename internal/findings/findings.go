// Package findings defines the output contract of reviewd:
// every detector produces Findings, and every consumer (CLI,
// eval scorer, GitHub Action) reads them
package findings

import "time"

// Severity of a finding
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
)

// Finding is a single defect reported by a detector.
type Finding struct {
	File        string   `json:"file"`
	Line        int      `json:"line"`
	Detector    string   `json:"detector"`
	RuleID      string   `json:"rule_id"`
	Severity    Severity `json:"severity"`
	Confidence  float64  `json:"confidence"`
	Explanation string   `json:"explanation"`
	Fingerprint string   `json:"fingerprint"`
}

// Report is the top-level output of `reviewd run`.
type Report struct {
	SchemaVersion int              `json:"schema_version"`
	Meta          Meta             `json:"meta"`
	Detectors     []DetectorStatus `json:"detectors"`
	Findings      []Finding        `json:"findings"`
}

type Meta struct {
	ReviewdVersion string    `json:"reviewd_version"`
	RepoSHA        string    `json:"repo_sha"`
	Ref            string    `json:"ref"`
	Model          string    `json:"model"`
	Threshold      float64   `json:"threshold"` // configured, not applied
	GeneratedAt    time.Time `json:"generated_at"`
}

type DetectorStatus struct {
	Name          string `json:"name"`
	PromptVersion string `json:"prompt_version,omitempty"`
	Status        string `json:"status"` // ran | errored | skipped
	Error         string `json:"error,omitempty"`
	DurationMS    int64  `json:"duration_ms"`
}
