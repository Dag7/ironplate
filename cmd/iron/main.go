// Package main is the entry point for the iron CLI.
package main

import (
	"fmt"
	"os"

	"github.com/dag7/ironplate/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
