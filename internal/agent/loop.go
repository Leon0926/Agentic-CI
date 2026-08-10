package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// ErrMaxIterations: the model never produced a final answer within
// budget. Sentinel so callers can errors.Is it — the io.EOF pattern.
var ErrMaxIterations = errors.New("agent: max iterations reached")

// LoopTool is the loop's own view of a tool. tools.Tool satisfies it
// structurally — this local interface is what breaks the import cycle.
type LoopTool interface {
	Def() ToolDef
	Run(ctx context.Context, input json.RawMessage) (string, error)
}

type LoopConfig struct {
	System    string
	Model     string
	MaxTokens int
	MaxIters  int
}

func RunLoop(ctx context.Context, client Client, cfg LoopConfig, initial string, tools []LoopTool) (string, error) {
	defs := make([]ToolDef, 0, len(tools))
	byName := make(map[string]LoopTool, len(tools))
	for _, t := range tools {
		d := t.Def()
		defs = append(defs, d)
		byName[d.Name] = t
	}

	msgs := []Message{{Role: RoleUser, Text: initial}}

	for i := 0; i < cfg.MaxIters; i++ {
		resp, err := client.Complete(ctx, Request{
			System: cfg.System, Messages: msgs, Tools: defs,
			Model: cfg.Model, MaxTokens: cfg.MaxTokens,
		})
		if err != nil {
			return "", err // infrastructure failure — abort
		}

		msgs = append(msgs, Message{
			Role: RoleAssistant, Text: resp.Text, ToolCalls: resp.ToolCalls,
		})

		if resp.StopReason != StopToolUse {
			return resp.Text, nil // model chose to finish
		}

		results := make([]ToolResult, 0, len(resp.ToolCalls))
		for _, call := range resp.ToolCalls {
			results = append(results, runOne(ctx, byName, call))
		}
		msgs = append(msgs, Message{Role: RoleUser, ToolResults: results})
	}
	return "", ErrMaxIterations
}

// runOne executes a single call, converting model-recoverable failures
// (unknown tool, tool-returned error) into IsError results instead of
// aborting the loop.
func runOne(ctx context.Context, byName map[string]LoopTool, call ToolCall) ToolResult {
	t, ok := byName[call.Name]
	if !ok {
		return ToolResult{CallID: call.ID, IsError: true,
			Content: fmt.Sprintf("no such tool: %s", call.Name)}
	}
	out, err := t.Run(ctx, call.Input)
	if err != nil {
		if ctx.Err() != nil {
			// context death isn't the model's fault; surface as content
			// anyway — next Complete will fail with the real error
			return ToolResult{CallID: call.ID, IsError: true, Content: ctx.Err().Error()}
		}
		return ToolResult{CallID: call.ID, IsError: true, Content: err.Error()}
	}
	return ToolResult{CallID: call.ID, Content: out}
}
