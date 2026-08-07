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
