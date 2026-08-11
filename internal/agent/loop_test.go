package agent

import (
	"context"
	"errors"
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
