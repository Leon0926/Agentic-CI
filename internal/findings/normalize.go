package findings

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// Normalize rewrites each Finding's File to be repo-relative, computes
// its Fingerprint, and sorts Findings deterministically. Called exactly
// once, right before a Report is encoded — see DESIGN.md. Detectors
// never do any of this themselves.
func (r *Report) Normalize(repoRoot string) error {
	root, err := filepath.Abs(repoRoot)
	if err != nil {
		return fmt.Errorf("findings: normalize: resolving repo root %q: %w", repoRoot, err)
	}

	for i := range r.Findings {
		f := &r.Findings[i]
		if rel, ok := relToRoot(root, f.File); ok {
			f.File = rel
		}
		// else: File is already relative, or absolute but outside root —
		// not this function's job to guess which; leave it as reported.
		f.Fingerprint = fingerprint(f)
	}

	sortFindings(r.Findings)
	return nil
}

// relToRoot returns path made relative to root, with forward slashes,
// when path is absolute and actually rooted under root. The second
// return is false (path left untouched by the caller) for a path that's
// already relative, or absolute but outside root.
func relToRoot(root, path string) (string, bool) {
	if !filepath.IsAbs(path) {
		return "", false
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

// fingerprint identifies "the same defect, at the same place, from the
// same rule" — deliberately built from Detector+RuleID+File+Line only.
// Explanation and Confidence can both drift between runs over an
// identical diff without that counting as a different finding; see
// DESIGN.md.
func fingerprint(f *Finding) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s\x00%s\x00%s\x00%d", f.Detector, f.RuleID, f.File, f.Line)
	return hex.EncodeToString(h.Sum(nil))
}

// sortFindings orders by file, then line, then detector — stable, so
// two runs over an identical diff with identical detector output
// produce byte-identical JSON.
func sortFindings(fs []Finding) {
	sort.SliceStable(fs, func(i, j int) bool {
		a, b := fs[i], fs[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Detector < b.Detector
	})
}
