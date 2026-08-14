package secrets

import (
	_ "embed"
)

// V1 is the system prompt for the LLM secrets detector, defining its
// tools, judging criteria, and required JSON output schema.
//
//go:embed prompts/v1.md
var V1 string
