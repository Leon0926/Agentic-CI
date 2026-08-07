package secrets

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Leon0926/Agentic-CI/internal/diff"
)

// helper: a single-file, single-hunk diff built by hand
// deliberately construct structs instead of parsing diff text:
// the parser has its own golden tests, and these tests should have exactly one reason to fail
func fileWith(lines []diff.Line) []diff.FileDiff {
	return []diff.FileDiff{
		{
			NewPath: "config/prod.go",
			Hunks: []diff.Hunk{
				{Lines: lines},
			},
		},
	}
}

func TestDetect(t *testing.T) {
	tests := []struct {
		name      string
		files     []diff.FileDiff
		wantCount int
		wantLine  int // only checked when wantCount == 1
	}{
		{
			name: "added AWS key is caught",
			files: fileWith([]diff.Line{
				{Op: diff.OpAdd, NewLine: 12, Content: `awsKey := "AKIAIOSFODNN7EXAMPLE"`},
			}),
			wantCount: 1,
			wantLine:  12,
		},
		{
			name: "AWS key on a context line is ignored",
			files: fileWith([]diff.Line{
				{Op: diff.OpContext, OldLine: 12, NewLine: 12, Content: `awsKey := "AKIAIOSFODNN7EXAMPLE"`},
			}),
			wantCount: 0,
		},
		{
			name: "env var password assignment is not a literal, ignored",
			files: fileWith([]diff.Line{
				{Op: diff.OpAdd, NewLine: 3, Content: `password := os.Getenv("DB_PASSWORD")`},
			}),
			wantCount: 0,
		},
		{
			name: "private key header caught at right line in multi-line hunk",
			files: fileWith([]diff.Line{
				{Op: diff.OpContext, OldLine: 1, NewLine: 1, Content: `package deploy`},
				{Op: diff.OpAdd, NewLine: 2, Content: ``},
				{Op: diff.OpAdd, NewLine: 3, Content: `const key = ` + "`" + `-----BEGIN RSA PRIVATE KEY-----`},
			}),
			wantCount: 1,
			wantLine:  3,
		},
		{
			name:      "empty diff yields no findings",
			files:     nil,
			wantCount: 0,
		},
	}

	d := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := d.Detect(context.Background(), tt.files)
			require.NoError(t, err)
			assert.Len(t, got, tt.wantCount)
			if tt.wantCount == 1 && len(got) == 1 {
				assert.Equal(t, tt.wantLine, got[0].Line)
				assert.Equal(t, "secrets", got[0].Detector)
				assert.Equal(t, "config/prod.go", got[0].File)
			}
		})
	}
}
