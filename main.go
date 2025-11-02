package main

import (
	"log/slog"
	"os"
	"os/exec"

	"github.com/davidsilvasanmartin/auto-homelab/cmd"
)

func main() {
	if err := requireCommand("docker"); err != nil {
		exitWithError("A command was not found. Is it installed in PATH?", "command", "docker")
	}
	if err := requireCommand("restic"); err != nil {
		exitWithError("A command was not found. Is it installed in PATH?", "command", "restic")
	}
	if err := cmd.Execute(); err != nil {
		// Cobra already prints the error; ensure non-zero exit for failure cases
		exitWithError("Command execution failed", "error", err.Error())
	}
}

func requireCommand(command string) error {
	_, err := exec.LookPath(command)
	return err
}

func exitWithError(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}
