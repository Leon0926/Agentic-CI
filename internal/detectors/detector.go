package detectors

import (
	"context"

	"github.com/Leon0926/Agentic-CI/internal/diff"
	"github.com/Leon0926/Agentic-CI/internal/findings"
)

// Detector examines a parsed diff and reports findings, must be safe to call with any diff, including empty
type Detector interface {
	// Name returns the stable identifier used in Finding.Detector and in config to enable/disable this detector
	Name() string

	// Detect walks the parsed diff and returns findings, must respect ctx cancellations once started
	Detect(ctx context.Context, files []diff.FileDiff) ([]findings.Finding, error)
}

// Versioned is implemented by detectors whose behavior is governed by a
// versioned prompt (LLM-backed detectors). A detector with no prompt —
// e.g. a pure-regex detector — simply doesn't implement this; callers
// type-assert and treat absence as an empty PromptVersion, so Detector
// itself stays at its two methods regardless of how many detectors ever
// need this.
type Versioned interface {
	PromptVersion() string
}
