package findings

import (
	"fmt"
	"strings"
)

// Severity of a finding. See ParseSeverity for the boundary where free
// text (model output, config) becomes one of these — nothing downstream
// should ever hold a Severity value outside this set.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
)

// ParseSeverity normalizes s (case-insensitive, surrounding whitespace
// trimmed) into one of the fixed Severity values. An unrecognized string
// is rejected rather than coerced to a default — see DESIGN.md.
func ParseSeverity(s string) (Severity, error) {
	switch Severity(strings.ToLower(strings.TrimSpace(s))) {
	case SeverityCritical:
		return SeverityCritical, nil
	case SeverityHigh:
		return SeverityHigh, nil
	case SeverityMedium:
		return SeverityMedium, nil
	case SeverityLow:
		return SeverityLow, nil
	default:
		return "", fmt.Errorf("findings: invalid severity %q", s)
	}
}
