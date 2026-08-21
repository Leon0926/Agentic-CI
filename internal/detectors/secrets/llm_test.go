package secrets

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Leon0926/Agentic-CI/internal/agent"
	"github.com/Leon0926/Agentic-CI/internal/diff"
	"github.com/Leon0926/Agentic-CI/internal/tools"
)

// fakeClient scripts model turns without ever calling the real API — same
// pattern as internal/agent/loop_test.go's fakeClient.
type fakeClient struct {
	queue []*agent.Response
	seen  []agent.Request
}

func (f *fakeClient) Complete(_ context.Context, req agent.Request) (*agent.Response, error) {
	f.seen = append(f.seen, req)
	if len(f.queue) == 0 {
		return nil, errors.New("fakeClient: queue exhausted — unexpected call to the model")
	}
	r := f.queue[0]
	f.queue = f.queue[1:]
	return r, nil
}

func toolUse(id, name, input string) agent.ToolCall {
	return agent.ToolCall{ID: id, Name: name, Input: json.RawMessage(input)}
}

// initGitFixture creates a real git repo so the real read_file/grep_repo
// tools have something to operate on — the tools aren't mocked, only the model is.
func initGitFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoErrorf(t, err, "git %v: %s", args, out)
	}
	run("init", "-q")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "test")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.go"),
		[]byte("package config\n\nvar AWSKey = \"AKIAIOSFODNN7EXAMPLE\" // test fixture, not a real key\n"), 0o644))
	run("add", "-A")
	run("commit", "-q", "-m", "seed")
	return dir
}

func TestLLMDetector_Detect(t *testing.T) {
	repo := initGitFixture(t)

	rf, err := tools.NewReadFile(repo)
	require.NoError(t, err)
	gr, err := tools.NewGrepRepo(repo)
	require.NoError(t, err)

	client := &fakeClient{queue: []*agent.Response{
		// turn 1: model wants to see the file before judging
		{
			StopReason: agent.StopToolUse,
			ToolCalls:  []agent.ToolCall{toolUse("call_1", "read_file", `{"path":"config.go"}`)},
		},
		// turn 2: model checks whether the key shows up elsewhere too
		{
			StopReason: agent.StopToolUse,
			ToolCalls:  []agent.ToolCall{toolUse("call_2", "grep_repo", `{"pattern":"AKIA[0-9A-Z]{16}"}`)},
		},
		// turn 3: model renders its verdict as the required JSON array
		{
			StopReason: agent.StopEnd,
			Text:       `[{"file":"config.go","line":3,"detector":"secrets-llm","severity":"high","confidence":0.9,"explanation":"hardcoded AWS key"}]`,
		},
	}}

	det := NewLLMDetector(client, []agent.LoopTool{rf, gr}, agent.LoopConfig{MaxIters: 5})

	files := []diff.FileDiff{{
		NewPath: "config.go",
		Hunks: []diff.Hunk{{Lines: []diff.Line{
			{Op: diff.OpAdd, NewLine: 3, Content: `var AWSKey = "AKIAIOSFODNN7EXAMPLE"`},
		}}},
	}}

	got, err := det.Detect(context.Background(), files)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "config.go", got[0].File)
	assert.Equal(t, 3, got[0].Line)
	assert.Equal(t, "secrets-llm", got[0].Detector) // stamped by parseFindings, not trusted from the model
	assert.Equal(t, 0.9, got[0].Confidence)

	// prove tool calling actually happened, for real, against the fixture repo
	require.Len(t, client.seen, 3)
	lastMsg := client.seen[1].Messages[len(client.seen[1].Messages)-1]
	require.Len(t, lastMsg.ToolResults, 1)
	assert.Contains(t, lastMsg.ToolResults[0].Content, "AKIAIOSFODNN7EXAMPLE") // real read_file output
	assert.False(t, lastMsg.ToolResults[0].IsError)
}

func TestLLMDetector_Detect_DropsFindingWithInvalidSeverity(t *testing.T) {
	client := &fakeClient{queue: []*agent.Response{
		{
			StopReason: agent.StopEnd,
			Text: `[` +
				`{"file":"config.go","line":3,"detector":"secrets-llm","severity":"urgent","confidence":0.9,"explanation":"bad severity, dropped"},` +
				`{"file":"config.go","line":5,"detector":"secrets-llm","severity":"high","confidence":0.7,"explanation":"valid, kept"}` +
				`]`,
		},
	}}

	det := NewLLMDetector(client, nil, agent.LoopConfig{MaxIters: 5})

	files := []diff.FileDiff{{
		NewPath: "config.go",
		Hunks: []diff.Hunk{{Lines: []diff.Line{
			{Op: diff.OpAdd, NewLine: 3, Content: `line 3`},
			{Op: diff.OpAdd, NewLine: 5, Content: `line 5`},
		}}},
	}}

	got, err := det.Detect(context.Background(), files)
	require.NoError(t, err)
	require.Len(t, got, 1, "the finding with an invalid severity should be dropped, not the whole response")
	assert.Equal(t, 5, got[0].Line)
}

func TestLLMDetector_Detect_EmptyDiffSkipsCallEntirely(t *testing.T) {
	client := &fakeClient{} // empty queue: any Complete() call fails the test
	det := NewLLMDetector(client, nil, agent.LoopConfig{MaxIters: 5})

	got, err := det.Detect(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, got)
	assert.Empty(t, client.seen, "no diff content means no reason to call the model at all")
}
