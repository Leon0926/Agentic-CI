package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
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
	// Logger receives structured entry/iteration/tool-call/exit events.
	// Callers can pre-bind context (detector name, file/line scope) via
	// slog.Logger.With before passing it in — every event below inherits
	// those attrs for free. Nil falls back to slog.Default() (stderr).
	Logger *slog.Logger
}

func RunLoop(ctx context.Context, client Client, cfg LoopConfig, initial string, tools []LoopTool) (string, error) {
	log := cfg.Logger
	if log == nil {
		log = slog.Default()
	}

	defs := make([]ToolDef, 0, len(tools))
	byName := make(map[string]LoopTool, len(tools))
	for _, t := range tools {
		d := t.Def()
		defs = append(defs, d)
		byName[d.Name] = t
	}

	log.Info("agent loop start",
		"model", cfg.Model, "max_iterations", cfg.MaxIters, "tools", len(tools))

	msgs := []Message{{Role: RoleUser, Text: initial}}

	for i := 0; i < cfg.MaxIters; i++ {
		resp, err := client.Complete(ctx, Request{
			System: cfg.System, Messages: msgs, Tools: defs,
			Model: cfg.Model, MaxTokens: cfg.MaxTokens,
		})
		if err != nil {
			log.Error("agent loop exit", "reason", "error", "iteration", i, "error", err)
			return "", err // infrastructure failure — abort
		}

		log.Info("agent loop iteration",
			"iteration", i,
			"stop_reason", resp.StopReason,
			"has_text", resp.Text != "",
			"tool_use_blocks", len(resp.ToolCalls),
		)

		msgs = append(msgs, Message{
			Role: RoleAssistant, Text: resp.Text, ToolCalls: resp.ToolCalls,
		})

		if resp.StopReason != StopToolUse {
			log.Info("agent loop exit", "reason", "final_text", "iteration", i)
			// debug only: tells you whether the model said nothing, or
			// said something the caller's parser then failed to eat.
			log.Debug("agent loop final text", "text", resp.Text)
			return resp.Text, nil // model chose to finish
		}

		results := make([]ToolResult, 0, len(resp.ToolCalls))
		for _, call := range resp.ToolCalls {
			results = append(results, runOne(ctx, log, byName, call))
		}
		msgs = append(msgs, Message{Role: RoleUser, ToolResults: results})
	}
	log.Warn("agent loop exit", "reason", "max_iterations", "iterations", cfg.MaxIters)
	return "", ErrMaxIterations
}

// runOne executes a single call, converting model-recoverable failures
// (unknown tool, tool-returned error) into IsError results instead of
// aborting the loop.
func runOne(ctx context.Context, log *slog.Logger, byName map[string]LoopTool, call ToolCall) ToolResult {
	t, ok := byName[call.Name]
	if !ok {
		log.Warn("tool call", "tool", call.Name, "error", "no such tool")
		return ToolResult{CallID: call.ID, IsError: true,
			Content: fmt.Sprintf("no such tool: %s", call.Name)}
	}
	out, err := t.Run(ctx, call.Input)
	if err != nil {
		errMsg := err.Error()
		if ctx.Err() != nil {
			// context death isn't the model's fault; surface as content
			// anyway — next Complete will fail with the real error
			errMsg = ctx.Err().Error()
		}
		log.Warn("tool call", "tool", call.Name, "args", string(call.Input), "error", errMsg)
		return ToolResult{CallID: call.ID, IsError: true, Content: errMsg}
	}
	log.Info("tool call", "tool", call.Name, "args", string(call.Input), "result_bytes", len(out))
	return ToolResult{CallID: call.ID, Content: out}
}
