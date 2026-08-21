package findings

import (
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalize_PathRewrite(t *testing.T) {
	root := filepath.FromSlash("/repo")

	r := &Report{Findings: []Finding{
		{File: filepath.FromSlash("/repo/internal/foo.go"), Line: 1, Detector: "secrets"},
	}}
	require.NoError(t, r.Normalize(root))
	assert.Equal(t, "internal/foo.go", r.Findings[0].File)
}

func TestNormalize_PathRewriteIsIdempotent(t *testing.T) {
	root := filepath.FromSlash("/repo")

	r := &Report{Findings: []Finding{
		{File: filepath.FromSlash("/repo/internal/foo.go"), Line: 1, Detector: "secrets"},
	}}
	require.NoError(t, r.Normalize(root))
	require.NoError(t, r.Normalize(root)) // already relative the second time
	assert.Equal(t, "internal/foo.go", r.Findings[0].File)
}

func TestNormalize_AlreadyRelativePathLeftAlone(t *testing.T) {
	r := &Report{Findings: []Finding{
		{File: "internal/foo.go", Line: 1, Detector: "secrets"},
	}}
	require.NoError(t, r.Normalize(filepath.FromSlash("/repo")))
	assert.Equal(t, "internal/foo.go", r.Findings[0].File)
}

func TestNormalize_PathOutsideRootLeftAlone(t *testing.T) {
	outside := filepath.FromSlash("/somewhere/else/foo.go")
	r := &Report{Findings: []Finding{
		{File: outside, Line: 1, Detector: "secrets"},
	}}
	require.NoError(t, r.Normalize(filepath.FromSlash("/repo")))
	assert.Equal(t, outside, r.Findings[0].File, "not this function's job to guess what an out-of-root path means")
}

func TestNormalize_FingerprintStableAcrossCosmeticFields(t *testing.T) {
	base := Finding{File: "internal/foo.go", Line: 12, Detector: "secrets", RuleID: "aws-access-key-id"}

	a := base
	a.Explanation = "first wording"
	a.Confidence = 0.9

	b := base
	b.Explanation = "reworded later"
	b.Confidence = 0.4

	r := &Report{Findings: []Finding{a, b}}
	require.NoError(t, r.Normalize(filepath.FromSlash("/repo")))

	assert.Equal(t, r.Findings[0].Fingerprint, r.Findings[1].Fingerprint,
		"fingerprint must not depend on Explanation or Confidence")
	assert.NotEmpty(t, r.Findings[0].Fingerprint)
}

func TestNormalize_FingerprintDiffersOnLine(t *testing.T) {
	r := &Report{Findings: []Finding{
		{File: "internal/foo.go", Line: 12, Detector: "secrets", RuleID: "aws-access-key-id"},
		{File: "internal/foo.go", Line: 13, Detector: "secrets", RuleID: "aws-access-key-id"},
	}}
	require.NoError(t, r.Normalize(filepath.FromSlash("/repo")))
	assert.NotEqual(t, r.Findings[0].Fingerprint, r.Findings[1].Fingerprint)
}

func TestNormalize_SortIsDeterministicRegardlessOfInputOrder(t *testing.T) {
	f1 := Finding{File: "a.go", Line: 5, Detector: "secrets"}
	f2 := Finding{File: "a.go", Line: 1, Detector: "secrets"}
	f3 := Finding{File: "b.go", Line: 1, Detector: "secrets"}
	f4 := Finding{File: "a.go", Line: 1, Detector: "secrets-llm"}

	r1 := &Report{Findings: []Finding{f1, f2, f3, f4}}
	r2 := &Report{Findings: []Finding{f4, f3, f2, f1}}

	require.NoError(t, r1.Normalize(filepath.FromSlash("/repo")))
	require.NoError(t, r2.Normalize(filepath.FromSlash("/repo")))

	assert.Equal(t, r1.Findings, r2.Findings)

	// explicit expected order: file asc, then line asc, then detector asc
	want := []string{"a.go:1:secrets", "a.go:1:secrets-llm", "a.go:5:secrets", "b.go:1:secrets"}
	var got []string
	for _, f := range r1.Findings {
		got = append(got, f.File+":"+strconv.Itoa(f.Line)+":"+f.Detector)
	}
	assert.Equal(t, want, got)
}
