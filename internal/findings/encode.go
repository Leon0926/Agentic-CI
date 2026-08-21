package findings

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// WriteJSON writes the full Report, unfiltered — every finding, at
// every confidence level, always. See DESIGN.md, "Confidence is not a
// filter". This is the machine contract; it never drops anything.
func WriteJSON(w io.Writer, r *Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(r); err != nil {
		return fmt.Errorf("findings: write json: %w", err)
	}
	return nil
}

// WriteText writes a human-readable view of r's findings, dropping any
// whose Confidence is below threshold. WriteJSON never filters — this
// is the only place a threshold is applied, by design; see DESIGN.md.
func WriteText(w io.Writer, r *Report, threshold float64) error {
	shown := 0
	for _, f := range r.Findings {
		if f.Confidence < threshold {
			continue
		}
		shown++

		ruleID := f.RuleID
		if ruleID == "" {
			ruleID = "-"
		}
		if _, err := fmt.Fprintf(w, "[%s] %s:%d  %s/%s  (confidence %.2f)\n    %s\n",
			strings.ToUpper(string(f.Severity)), f.File, f.Line, f.Detector, ruleID, f.Confidence, f.Explanation,
		); err != nil {
			return fmt.Errorf("findings: write text: %w", err)
		}
	}

	total := len(r.Findings)
	if _, err := fmt.Fprintf(w, "\n%d finding(s) shown (%d below confidence threshold %.2f)\n",
		shown, total-shown, threshold,
	); err != nil {
		return fmt.Errorf("findings: write text: %w", err)
	}
	return nil
}
