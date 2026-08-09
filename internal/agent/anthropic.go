package agent

import (
	"context"
	"encoding/json"
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

// helper func to turn []any into []string, used for ToolDef.InputSchema.Required
func toStringSlice(v any) []string {
	raw, ok := v.([]string)
	if ok {
		return raw
	}
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		s, ok := item.(string)
		if !ok {
			continue
		}
		out = append(out, s)
	}
	return out
}

// toSDKRequest maps Messages (text / tool calls / tool results) into
// SDK content blocks, and ToolDefs into SDK tool params.
func toSDKRequest(req Request) (anthropic.MessageNewParams, error) {
	// ... loop over req.Messages, build content blocks per role;
	// text → text block, ToolCalls → tool_use blocks,
	// ToolResults → tool_result blocks. Tools from req.Tools.
	sdkMessages := make([]anthropic.MessageParam, 0, len(req.Messages))

	for _, m := range req.Messages {
		var blocks []anthropic.ContentBlockParamUnion

		if m.Text != "" {
			blocks = append(blocks, anthropic.NewTextBlock(m.Text))
		}

		for _, tc := range m.ToolCalls {
			var input any
			if err := json.Unmarshal(tc.Input, &input); err != nil {
				return anthropic.MessageNewParams{}, fmt.Errorf("agent: decode tool call input: %w", err)
			}
			blocks = append(blocks, anthropic.ContentBlockParamUnion{
				OfToolUse: &anthropic.ToolUseBlockParam{
					ID:    tc.ID,
					Name:  tc.Name,
					Input: input,
				},
			})
		}

		for _, tr := range m.ToolResults {
			blocks = append(blocks, anthropic.NewToolResultBlock(tr.CallID, tr.Content, tr.IsError))
		}

		role := anthropic.MessageParamRoleUser
		if m.Role == RoleAssistant {
			role = anthropic.MessageParamRoleAssistant
		}
		sdkMessages = append(sdkMessages, anthropic.MessageParam{Role: role, Content: blocks})
	}

	sdkTools := make([]anthropic.ToolUnionParam, 0, len(req.Tools))
	for _, t := range req.Tools {
		sdkTools = append(sdkTools, anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        t.Name,
				Description: anthropic.String(t.Description),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: t.InputSchema["properties"],
					Required:   toStringSlice(t.InputSchema["required"]),
				},
			},
		})
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(req.Model),
		MaxTokens: int64(req.MaxTokens),
		Messages:  sdkMessages,
		Tools:     sdkTools,
	}
	if req.System != "" {
		params.System = []anthropic.TextBlockParam{{Text: req.System}}
	}
	return params, nil
}

func fromSDKResponse(msg *anthropic.Message) *Response {
	// fromSDKResponse walks msg.Content: text blocks concatenate into
	// Text, tool_use blocks become ToolCalls. Stop reason maps to our enum:
	// "end_turn" → StopEnd, "tool_use" → StopToolUse, else StopOther.
	resp := &Response{}

	for _, block := range msg.Content {
		switch b := block.AsAny().(type) {
		case anthropic.TextBlock:
			resp.Text += b.Text
		case anthropic.ToolUseBlock:
			resp.ToolCalls = append(resp.ToolCalls, ToolCall{
				ID:    b.ID,
				Name:  b.Name,
				Input: json.RawMessage(b.Input),
			})
		}
	}

	switch msg.StopReason {
	case anthropic.StopReasonEndTurn:
		resp.StopReason = StopEnd
	case anthropic.StopReasonToolUse:
		resp.StopReason = StopToolUse
	default:
		resp.StopReason = StopOther
	}

	return resp
}
