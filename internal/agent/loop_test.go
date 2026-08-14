package agent

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// fakeClient pops responses off a queue and records every Request
// it received, so tests can assert what the loop actually sent.
type fakeClient struct {
	queue []*Response
	seen  []Request
}

func (f *fakeClient) Complete(_ context.Context, req Request) (*Response, error) {
	f.seen = append(f.seen, req)
	if len(f.queue) == 0 {
		return nil, errors.New("fake: queue exhausted")
	}
	r := f.queue[0]
	f.queue = f.queue[1:]
	return r, nil
}

// fakeTool records calls and returns a canned string (or error).

// Table-driven cases:
//   "happy path"      — queue: [tool_use(read_file), StopEnd("done")]
//                       → returns "done"; fakeTool called once;
//                       seen[1] contains an IsError=false ToolResult
//   "max iterations"  — queue: tool_use × N+1, MaxIters: N
//                       → errors.Is(err, ErrMaxIterations)
//   "unknown tool"    — queue: [tool_use("nope"), StopEnd("ok")]
//                       → no error; seen[1] has IsError result
//                       containing "no such tool"
//   "tool errors"     — fakeTool returns error
//                       → loop continues; IsError result fed back
//   "client failure"  — fake returns error → RunLoop aborts with it

type fakeTool struct {
	name   string
	out    string
	err    error
	called int
}

func (f *fakeTool) Def() ToolDef { return ToolDef{Name: f.name} }

func (f *fakeTool) Run(_ context.Context, _ json.RawMessage) (string, error) {
	f.called++
	return f.out, f.err
}

// --- helpers ---

func toolUseResp(name string) *Response {
	return &Response{
		StopReason: StopToolUse,
		ToolCalls:  []ToolCall{{ID: "call_1", Name: name, Input: json.RawMessage(`{}`)}},
	}
}

func endResp(text string) *Response {
	return &Response{StopReason: StopEnd, Text: text}
}

func repeat(r *Response, n int) []*Response {
	out := make([]*Response, n)
	for i := range out {
		out[i] = r
	}
	return out
}

// --- the tests ---

func TestRunLoop(t *testing.T) {
	cfg := LoopConfig{MaxIters: 3}

	t.Run("happy path", func(t *testing.T) {
		tool := &fakeTool{name: "read_file", out: "file contents"}
		client := &fakeClient{queue: []*Response{
			toolUseResp("read_file"),
			endResp("done"),
		}}

		got, err := RunLoop(context.Background(), client, cfg, "go", []LoopTool{tool})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "done" {
			t.Errorf("final text = %q, want %q", got, "done")
		}
		if tool.called != 1 {
			t.Errorf("tool called %d times, want 1", tool.called)
		}

		// The invariant: the result actually reached the second request,
		// paired to the right call, not flagged as error.
		req2 := client.seen[1]
		last := req2.Messages[len(req2.Messages)-1]
		if len(last.ToolResults) != 1 {
			t.Fatalf("expected 1 tool result in second request, got %d", len(last.ToolResults))
		}
		tr := last.ToolResults[0]
		if tr.CallID != "call_1" || tr.IsError || tr.Content != "file contents" {
			t.Errorf("tool result = %+v, want CallID=call_1, IsError=false, Content=%q",
				tr, "file contents")
		}
	})

	t.Run("max iterations", func(t *testing.T) {
		tool := &fakeTool{name: "read_file", out: "x"}
		client := &fakeClient{queue: repeat(toolUseResp("read_file"), cfg.MaxIters+1)}

		_, err := RunLoop(context.Background(), client, cfg, "go", []LoopTool{tool})
		if !errors.Is(err, ErrMaxIterations) {
			t.Errorf("err = %v, want ErrMaxIterations", err)
		}
		if len(client.seen) != cfg.MaxIters {
			t.Errorf("client called %d times, want exactly %d", len(client.seen), cfg.MaxIters)
		}
	})

	t.Run("unknown tool", func(t *testing.T) {
		// no tools registered at all — every call is unknown
		client := &fakeClient{queue: []*Response{
			toolUseResp("nope"),
			endResp("ok"),
		}}

		got, err := RunLoop(context.Background(), client, cfg, "go", nil)
		if err != nil {
			t.Fatalf("loop should survive unknown tool, got: %v", err)
		}
		if got != "ok" {
			t.Errorf("final text = %q, want %q", got, "ok")
		}
		tr := client.seen[1].Messages[len(client.seen[1].Messages)-1].ToolResults[0]
		if !tr.IsError || !strings.Contains(tr.Content, "no such tool") {
			t.Errorf("tool result = %+v, want IsError with 'no such tool'", tr)
		}
	})

	t.Run("tool error fed back", func(t *testing.T) {
		tool := &fakeTool{name: "read_file", err: errors.New("file not found")}
		client := &fakeClient{queue: []*Response{
			toolUseResp("read_file"),
			endResp("recovered"),
		}}

		got, err := RunLoop(context.Background(), client, cfg, "go", []LoopTool{tool})
		if err != nil {
			t.Fatalf("tool error must not abort the loop: %v", err)
		}
		if got != "recovered" {
			t.Errorf("final text = %q, want %q", got, "recovered")
		}
		tr := client.seen[1].Messages[len(client.seen[1].Messages)-1].ToolResults[0]
		if !tr.IsError || tr.Content != "file not found" {
			t.Errorf("tool result = %+v, want IsError with the tool's message", tr)
		}
	})

	t.Run("client failure aborts", func(t *testing.T) {
		client := &fakeClient{} // empty queue → Complete errors immediately
		_, err := RunLoop(context.Background(), client, cfg, "go", nil)
		if err == nil {
			t.Fatal("expected error from client failure")
		}
		if errors.Is(err, ErrMaxIterations) {
			t.Error("client failure must not masquerade as max-iterations")
		}
	})
}
