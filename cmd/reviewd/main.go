package main

import (
	"fmt"
	"os"

	"github.com/Agentic-CI/agentic-ci/cmd/reviewd/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "reviewd", err)
		os.Exit(1)
	}
}
