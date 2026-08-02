// Package findings defines the output contract of reviewd:
// every detector produces Findings, and every consumer (CLI,
// eval scorer, GitHub Action) reads them
package findings

// Severity of a finding
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
)

// Finding is a single defect reported by a detector
type Finding struct {
	File        string   `json:"file"`
	Line        int      `json:"line"`
	Detector    string   `json:"detector"`
	Severity    Severity `json:"severity"`
	Confidence  float64  `json:"confidence"`
	Explanation string   `json:"explanation"`
}

// Report is the top-level output of `reviewd run`.
type Report struct {
	Findings []Finding `json:"findings"`
}
