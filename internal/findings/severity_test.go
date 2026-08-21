package findings

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSeverity(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    Severity
		wantErr bool
	}{
		{name: "lowercase", in: "high", want: SeverityHigh},
		{name: "mixed case", in: "High", want: SeverityHigh},
		{name: "uppercase", in: "CRITICAL", want: SeverityCritical},
		{name: "surrounding whitespace trimmed", in: "  medium  ", want: SeverityMedium},
		{name: "low", in: "low", want: SeverityLow},
		{name: "unrecognized value is rejected, not coerced", in: "urgent", wantErr: true},
		{name: "empty string is rejected", in: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSeverity(tt.in)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, got)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
