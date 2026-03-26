// Package main is the entry point for the iron CLI.
package main

import (
	"os"

	"github.com/dag7/ironplate/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
