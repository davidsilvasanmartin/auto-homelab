package main

import (
	"log/slog"
	"os"

	"auto-homelab/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		// Cobra already prints the error; ensure non-zero exit for failure cases
		slog.Error("Command execution failed", "error", err)
		os.Exit(1)
	}
}
