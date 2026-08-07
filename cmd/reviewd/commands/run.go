package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/Leon0926/Agentic-CI/internal/detectors"
	"github.com/Leon0926/Agentic-CI/internal/detectors/secrets"
	"github.com/Leon0926/Agentic-CI/internal/diff"
	"github.com/Leon0926/Agentic-CI/internal/findings"
	"github.com/spf13/cobra"
)

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

		// parse raw into hunks -> internal/diff -> DONE!!!
		files, err := diff.Parse(strings.NewReader(raw))
		if err != nil {
			return fmt.Errorf("parsing diff: %w", err)
		}

		// run secrets detector loop  -> internal/detectors/secrets -> DONE!!!

		dets := []detectors.Detector{
			secrets.New(),
		}

		report := findings.Report{Findings: []findings.Finding{}}
		ctx := cmd.Context() // cobra threads a context through; use it now so step 4 is free

		for _, det := range dets {
			found, err := det.Detect(ctx, files)
			if err != nil {
				return fmt.Errorf("detector %s: %w", det.Name(), err)
			}
			report.Findings = append(report.Findings, found...)
		}

		// create disposable worktree -> internal/sandbox
		// do this later
		// run other detectors         -> internal/detectors/...
		// collect findings            -> internal/findings

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	},
}

func init() {
	runCmd.Flags().String("diff", "", "git diff range to review (e.g. origin/main...HEAD)")
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
	// no diff provided, return empty string and no error
	return "", nil

}
