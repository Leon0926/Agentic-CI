package commands

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Leon0926/Agentic-CI/internal/agent"
	"github.com/Leon0926/Agentic-CI/internal/detectors"
	"github.com/Leon0926/Agentic-CI/internal/detectors/secrets"
	"github.com/Leon0926/Agentic-CI/internal/diff"
	"github.com/Leon0926/Agentic-CI/internal/findings"
	"github.com/Leon0926/Agentic-CI/internal/sandbox"
	"github.com/Leon0926/Agentic-CI/internal/tools"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var errNoDiffSource = errors.New("no diff provided: pass --diff <range> or pipe a diff on stdin")

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "run detector agents and produce a report",
	Long: `reads a diff and runs enabled detetor agents to produce a report of findings
	
Diff sources, in priority order:
  --diff <range>   e.g. --diff origin/main...HEAD (runs git diff <range>)
  stdin            pipe a diff in: git diff | reviewd run`,

	RunE: func(cmd *cobra.Command, args []string) error {
		raw, err := readDiff(cmd)
		if err != nil {
			return err
		}

		files, err := diff.Parse(strings.NewReader(raw))
		if err != nil {
			return fmt.Errorf("parsing diff: %w", err)
		}

		ctx := cmd.Context()
		repoPath, _ := cmd.Flags().GetString("repo")
		ref, _ := cmd.Flags().GetString("ref")

		// --- assembly: build the full detector list first ---

		dets := []detectors.Detector{secrets.New()}

		if viper.GetBool("detectors.secrets_llm.enabled") {
			wt, err := sandbox.New(ctx, repoPath, ref)
			if err != nil {
				return fmt.Errorf("sandbox: %w", err)
			}
			defer wt.Close()

			rf, err := tools.NewReadFile(wt.Root)
			if err != nil {
				return err
			}
			gr, err := tools.NewGrepRepo(wt.Root)
			if err != nil {
				return err
			}

			dets = append(dets, secrets.NewLLMDetector(
				agent.NewAnthropicClient(),
				[]agent.LoopTool{rf, gr},
				agent.LoopConfig{
					System:    secrets.V1,
					Model:     viper.GetString("model"),
					MaxTokens: viper.GetInt("max_tokens"),
					MaxIters:  viper.GetInt("max_iterations"),
					Logger:    Logger,
				},
			))
		}

		// --- execution: run every detector, merge findings. One detector
		// erroring is recorded on its DetectorStatus, not fatal to the run —
		// the other detectors' findings still make it into the report. ---

		threshold := viper.GetFloat64("confidence-threshold")
		report := findings.Report{
			SchemaVersion: 1,
			Meta: findings.Meta{
				ReviewdVersion: Version,
				RepoSHA:        repoSHA(repoPath),
				Ref:            ref,
				Model:          viper.GetString("model"),
				Threshold:      threshold,
				GeneratedAt:    time.Now().UTC(),
			},
			Findings: []findings.Finding{},
		}
		for _, det := range dets {
			start := time.Now()
			status := findings.DetectorStatus{Name: det.Name()}
			if v, ok := det.(detectors.Versioned); ok {
				status.PromptVersion = v.PromptVersion()
			}

			found, err := det.Detect(ctx, files)
			status.DurationMS = time.Since(start).Milliseconds()

			if err != nil {
				status.Status = "errored"
				status.Error = err.Error()
				Logger.Warn("detector failed", "detector", det.Name(), "error", err)
			} else {
				status.Status = "ran"
				report.Findings = append(report.Findings, found...)
			}
			report.Detectors = append(report.Detectors, status)
		}

		if err := report.Normalize(repoPath); err != nil {
			return fmt.Errorf("normalizing report: %w", err)
		}

		// findings are the only thing that ever go to stdout — progress and
		// warnings above all went through Logger, which is stderr-only. See
		// root.go's comment on Logger for why that split matters.
		out := cmd.OutOrStdout()
		format, _ := cmd.Flags().GetString("format")
		switch format {
		case "json":
			err = findings.WriteJSON(out, &report)
		case "text":
			err = findings.WriteText(out, &report, threshold)
		default:
			err = fmt.Errorf("unknown --format %q (want \"text\" or \"json\")", format)
		}
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	runCmd.Flags().String("diff", "", "git diff range to review (e.g. origin/main...HEAD)")
	runCmd.Flags().String("repo", ".", "path to the git repository")
	runCmd.Flags().String("ref", "HEAD", "git ref to create the review worktree from")
	runCmd.Flags().String("format", "text", `output format: "text" or "json"`)
	rootCmd.AddCommand(runCmd)
}

// repoSHA returns the current commit SHA of the repo at repoPath, or ""
// if it can't be resolved (e.g. not a git repo, no commits yet). Best
// effort and deliberately non-fatal — Meta.RepoSHA is informational, not
// load-bearing for the report itself.
func repoSHA(repoPath string) string {
	out, err := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func readDiff(cmd *cobra.Command) (string, error) {
	// read diff from --diff flag or stdin
	// check if flag is set and run git diff <range>
	if rng, _ := cmd.Flags().GetString("diff"); rng != "" {
		out, err := exec.Command("git", "diff", rng).Output()
		if err != nil {
			return "", fmt.Errorf("git diff %s: %w", rng, err)
		}
		// return the diff output as string
		return string(out), nil
	}
	// check if stdin is not a terminal (keyboard)
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 { // data is being piped in (not terminal so ==0)
		b, err := io.ReadAll(os.Stdin)
		// some other error reading stdin, return it
		if err != nil {
			return "", fmt.Errorf("reading stdin: %w", err)
		}
		return string(b), nil
	}
	// no diff provided, return no diff error
	return "", errNoDiffSource

}
