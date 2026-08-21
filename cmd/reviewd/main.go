package main

import (
	"fmt"
	"os"

	"github.com/Leon0926/Agentic-CI/cmd/reviewd/commands"
)

func main() {
	err := commands.Execute()
	if err != nil {
		// rootCmd has SilenceErrors set, so nothing has printed this yet.
		fmt.Fprintln(os.Stderr, "reviewd:", err)
	}
	os.Exit(commands.ExitCode(err))
}
