// Package secrets implements a credential-leak detector, currently pure regex over added lines
// Pattern table later becomes pre-filter infront of llm judge
package secrets

import (
	"context"
	"regexp"

	"github.com/Leon0926/Agentic-CI/internal/diff"
	"github.com/Leon0926/Agentic-CI/internal/findings"
)

// pattern pairs a compiled regex with the metadata reused in every Finding
type pattern struct {
	name        string
	re          *regexp.Regexp
	explanation string
}

// MustCompile checks for a malformed patterns at startup
var patterns = []pattern{
	{
		name:        "aws-access-key-id",
		re:          regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
		explanation: "Possible AWS access key ID committed in the diff.",
	},
	{
		name:        "private-key-header",
		re:          regexp.MustCompile(`-----BEGIN (RSA |EC |OPENSSH )?PRIVATE KEY-----`),
		explanation: "Private key material appears to be committed in the diff.",
	},
	{
		name:        "generic-credential-assignment",
		re:          regexp.MustCompile(`(?i)(api[_-]?key|secret|token|password)\s*[:=]\s*["'][^"']{8,}["']`),
		explanation: "Hardcoded credential assigned to a string literal.",
	},
}

// Detector is the secrets detector; stateless for now. may hold config/cache of seen findings later
type Detector struct{}

// constructor returns a secrets Detector
func New() *Detector {
	return &Detector{}
}

// Name implements detectors.Detector.
func (d *Detector) Name() string {
	return "secrets"
}

// Detect scans only added lines, context lines are ignored/outside of PR diff
func (d *Detector) Detect(ctx context.Context, files []diff.FileDiff) ([]findings.Finding, error) {
	var out []findings.Finding

	for _, f := range files {
		for _, h := range f.Hunks {
			for _, ln := range h.Lines {
				if ln.Op != diff.OpAdd { // only scan added lines
					continue
				}
				for _, p := range patterns {
					if p.re.MatchString(ln.Content) { // match found, add a finding
						out = append(out, findings.Finding{
							File:        f.NewPath, //
							Line:        ln.NewLine,
							Detector:    d.Name(),
							Severity:    findings.SeverityHigh,
							Confidence:  0.9, // fixed until the LLM judge exists (step 4)
							Explanation: p.explanation,
						})
					}
				}
			}
		}
	}

	return out, nil
}
