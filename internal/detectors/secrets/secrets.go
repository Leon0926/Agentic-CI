// Package secrets implements a credential-leak detector, currently pure regex over added lines
// Pattern table later becomes pre-filter infront of llm judge
package secrets

import (
	"context"
	"regexp"

	"github.com/Leon0926/Agentic-CI/internal/diff"
	"github.com/Leon0926/Agentic-CI/internal/findings"
)

// pattern pairs a compiled regex with the metadata reused in every Finding.
type pattern struct {
	name        string
	re          *regexp.Regexp
	explanation string
}

// MustCompile is correct here: a malformed pattern is a programmer error,
// and we want a loud panic at startup, not a silent skip at scan time.
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

// Detector is the secrets detector. Stateless for now.
type Detector struct{}

// New returns a secrets Detector.
func New() *Detector {
	return &Detector{}
}

// Name implements detectors.Detector.
func (d *Detector) Name() string {
	return "secrets"
}

// Detect scans only added lines: flagging context lines would mean
// flagging code the PR didn't touch, which is pure noise.
func (d *Detector) Detect(ctx context.Context, files []diff.FileDiff) ([]findings.Finding, error) {
	var out []findings.Finding

	for _, f := range files {
		for _, h := range f.Hunks {
			for _, ln := range h.Lines {
				if ln.Op != diff.OpAdd { // adapt to your Op constant name
					continue
				}
				for _, p := range patterns {
					if p.re.MatchString(ln.Content) { // adapt to your content field name
						out = append(out, findings.Finding{
							File:        f.NewPath, // new-file path: correct for renames
							Line:        ln.NewLine,
							Detector:    d.Name(),
							Severity:    findings.SeverityHigh, // adapt to your enum value
							Confidence:  0.9,                   // fixed until the LLM judge exists (step 4)
							Explanation: p.explanation,
						})
					}
				}
			}
		}
	}

	return out, nil
}
