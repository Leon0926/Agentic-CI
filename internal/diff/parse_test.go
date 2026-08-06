package diff

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "rewrite golden files")

func TestParse(t *testing.T) {
	inputs, err := filepath.Glob("testdata/*.diff")
	require.NoError(t, err)
	require.NotEmpty(t, inputs, "no testdata found")

	for _, in := range inputs {
		t.Run(filepath.Base(in), func(t *testing.T) {
			f, err := os.Open(in)
			require.NoError(t, err)
			defer f.Close()

			got, err := Parse(f)
			require.NoError(t, err)

			gotJSON, err := json.MarshalIndent(got, "", "  ")
			require.NoError(t, err)

			golden := strings.TrimSuffix(in, ".diff") + ".golden.json"

			if *update {
				require.NoError(t, os.WriteFile(golden, gotJSON, 0o644))
				return
			}

			want, err := os.ReadFile(golden)
			require.NoError(t, err, "golden file missing — run: go test ./internal/diff -update")
			assert.JSONEq(t, string(want), string(gotJSON))
		})
	}
}
