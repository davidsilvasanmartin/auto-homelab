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
	if err := requireCommand("sh"); err != nil {
		exitWithError("A command was not found. Is it installed in PATH?", "command", "sh")
	}
	if err := requireCommand("cp"); err != nil {
		// In this project we call the system's "cp" command to copy files or directories. This is just lazy
		// of me because I don't want to maintain the code required to do this. In the future we can
		// use a library or write custom code to handle this scenario
		exitWithError("A command was not found. Is it installed in PATH?", "command", "cp")
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
