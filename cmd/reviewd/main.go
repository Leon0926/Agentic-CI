package main

import (
	"fmt"
	"os"

	"github.com/Leon0926/Agentic-CI/cmd/reviewd/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "reviewd", err)
		os.Exit(1)
	}
}
