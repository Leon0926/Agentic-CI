package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Leon0926/Agentic-CI/internal/agent"
	"github.com/Leon0926/Agentic-CI/internal/diff"
	"github.com/Leon0926/Agentic-CI/internal/findings"
)

type LLMDetector struct {
	client agent.Client
	tools  []agent.LoopTool
	cfg    agent.LoopConfig // System prompt loaded from prompts/v1.md
}

func NewLLMDetector(client agent.Client, tools []agent.LoopTool, cfg agent.LoopConfig) *LLMDetector {
	return &LLMDetector{client: client, tools: tools, cfg: cfg}
}

func (d *LLMDetector) Name() string { return "secrets-llm" }

func (d *LLMDetector) Detect(ctx context.Context, files []diff.FileDiff) ([]findings.Finding, error) {
	added := countAddedLines(files)
	if added == 0 {
		return nil, nil // nothing new to review — don't spend a model call on it
	}

	// Pre-bind detector/scope attrs so every loop.go log line (entry,
	// iteration, tool call, exit) carries them for free via slog.With.
	log := d.logger().With(
		"detector", d.Name(),
		"files", len(files),
		"added_lines", added,
	)
	cfg := d.cfg
	cfg.Logger = log

	initial := renderDiff(files) // paths, hunks, added lines with numbers
	final, err := agent.RunLoop(ctx, d.client, cfg, initial, d.tools)
	if err != nil {
		return nil, fmt.Errorf("secrets-llm: %w", err)
	}

	fs, err := parseFindings(final)
	if err != nil {
		return nil, fmt.Errorf("secrets-llm: %w", err)
	}
	log.Info("secrets-llm detect finished", "findings", len(fs))
	return fs, nil
}

// logger falls back to slog.Default() (stderr) so callers that build an
// LLMDetector without a Logger — tests, mainly — never panic.
func (d *LLMDetector) logger() *slog.Logger {
	if d.cfg.Logger != nil {
		return d.cfg.Logger
	}
	return slog.Default()
}

// countAddedLines is a scope proxy for logging: the diff can span
// multiple files/hunks, so this is the closest single number to "how
// much" got sent to the model, in place of a single file/line range.
func countAddedLines(files []diff.FileDiff) int {
	n := 0
	for _, f := range files {
		for _, h := range f.Hunks {
			n += len(h.AddedLines())
		}
	}
	return n
}

// renderDiff produces the model-facing view of the diff: one section
// per file, added lines prefixed with their new-file line number.
// Line numbers here are load-bearing — the model cites them back.
func renderDiff(files []diff.FileDiff) string {
	var b strings.Builder
	for _, f := range files {
		var added []diff.Line
		for _, h := range f.Hunks {
			added = append(added, h.AddedLines()...)
		}
		if len(added) == 0 {
			continue // nothing new in this file, skip the section
		}
		fmt.Fprintf(&b, "--- %s\n", f.NewPath)
		for _, ln := range added {
			fmt.Fprintf(&b, "+%d: %s\n", ln.NewLine, ln.Content)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// parseFindings strips ```json fences defensively, unmarshals into
// []findings.Finding, and validates detector/severity fields.
// Unparseable output is an error we surface — for now. When the eval
// harness exists, "switch to an emit_findings tool" becomes a
// measured experiment instead of a guess.
func parseFindings(text string) ([]findings.Finding, error) {
	cleaned := stripFences(strings.TrimSpace(text))
	var fs []findings.Finding
	if err := json.Unmarshal([]byte(cleaned), &fs); err != nil {
		return nil, fmt.Errorf("secrets-llm: unparseable findings: %w; raw output: %.200s", err, text)
	}
	for i := range fs {
		fs[i].Detector = "secrets-llm" // stamp, don't trust the model
	}
	return fs, nil
}

// stripFences removes a leading/trailing ```json (or bare ```) fence in
// case the model wraps its array despite being told to reply with only
// the JSON array.
func stripFences(s string) string {
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
