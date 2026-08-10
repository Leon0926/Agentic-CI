package tools

import (
	"context"
	"encoding/json"

	"github.com/Leon0926/Agentic-CI/internal/agent"
)

type Tool interface {
	Def() agent.ToolDef
	Run(ctx context.Context, input json.RawMessage) (string, error)
}
