package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

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

		// --- assembly: build the full detector list first ---

		dets := []detectors.Detector{secrets.New()}

		if viper.GetBool("detectors.secrets_llm.enabled") {
			repoPath, _ := cmd.Flags().GetString("repo")
			ref, _ := cmd.Flags().GetString("ref")

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
				},
			))
		}

		// --- execution: run every detector, merge findings ---

		report := findings.Report{Findings: []findings.Finding{}}
		for _, det := range dets {
			found, err := det.Detect(ctx, files)
			if err != nil {
				return fmt.Errorf("detector %s: %w", det.Name(), err)
			}
			report.Findings = append(report.Findings, found...)
		}

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	},
}

func init() {
	runCmd.Flags().String("diff", "", "git diff range to review (e.g. origin/main...HEAD)")
	runCmd.Flags().String("repo", ".", "path to the git repository")
	runCmd.Flags().String("ref", "HEAD", "git ref to create the review worktree from")
	rootCmd.AddCommand(runCmd)
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
