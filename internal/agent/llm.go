package agent

import (
	"context"
	"encoding/json"
)

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type StopReason string

const (
	StopEnd     StopReason = "end"      // model finished its answer
	StopToolUse StopReason = "tool_use" // model wants tools run
	StopOther   StopReason = "other"    // other reason to stop e.g. max tokens reached
)

// Assistant turns may carry ToolCalls
// User turns may carry ToolResults answering them
type Message struct {
	Role        Role
	Text        string
	ToolCalls   []ToolCall
	ToolResults []ToolResult
}

// ToolCall is the model asking for a tool to run
// Input is json, agent routes it and only tool knows how to parse it
type ToolCall struct {
	ID    string
	Name  string
	Input json.RawMessage
}

// ToolResult answers one ToolCall. IsError means the model made a recoverable error,
// not that the tool failed; agent may retry
type ToolResult struct {
	CallID  string
	Content string
	IsError bool
}

// ToolDef is what the model is told a tool looks like
type ToolDef struct {
	Name        string
	Description string
	InputSchema map[string]any // JSON Schema
}

type Request struct {
	System    string
	Messages  []Message
	Tools     []ToolDef
	Model     string
	MaxTokens int
}

type Response struct {
	Text       string
	ToolCalls  []ToolCall
	StopReason StopReason
}

// decouples agent from LLM SDK (request -> response), anthropic.go implmement this
// loop_test uses a fake impl to test agent logic without calling real API
type Client interface {
	Complete(ctx context.Context, req Request) (*Response, error)
}
