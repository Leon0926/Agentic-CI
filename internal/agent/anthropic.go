package agent

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
)

type AnthropicClient struct {
	sdk anthropic.Client
}

// NewAnthropicClient reads ANTHROPIC_API_KEY from the environment
// (the SDK does this by default). Model/tokens come per-Request,
// so config plumbing stays in cmd/.
func NewAnthropicClient() *AnthropicClient {
	return &AnthropicClient{sdk: anthropic.NewClient()}
}

func (c *AnthropicClient) Complete(ctx context.Context, req Request) (*Response, error) {
	sdkReq, err := toSDKRequest(req)
	if err != nil {
		return nil, fmt.Errorf("agent: build request: %w", err)
	}
	msg, err := c.sdk.Messages.New(ctx, sdkReq)
	if err != nil {
		return nil, fmt.Errorf("agent: anthropic call: %w", err)
	}
	return fromSDKResponse(msg), nil
}

// toSDKRequest maps Messages (text / tool calls / tool results) into
// SDK content blocks, and ToolDefs into SDK tool params.
func toSDKRequest(req Request) (anthropic.MessageNewParams, error) {
	// ... loop over req.Messages, build content blocks per role;
	// text → text block, ToolCalls → tool_use blocks,
	// ToolResults → tool_result blocks. Tools from req.Tools.
}

// fromSDKResponse walks msg.Content: text blocks concatenate into
// Text, tool_use blocks become ToolCalls. Stop reason maps to our enum:
// "end_turn" → StopEnd, "tool_use" → StopToolUse, else StopOther.
func fromSDKResponse(msg *anthropic.Message) *Response {
	// ...
}
