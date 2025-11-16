package main

import (
	"log/slog"
	"os"
	"os/exec"

	"github.com/davidsilvasanmartin/auto-homelab/cmd"
	"github.com/davidsilvasanmartin/auto-homelab/internal/dotenv"
)

// TODO
//  - Redacting passwords in logs
//  - Unifying logs (the ones from dotenv/loader.go come in a different format)
//  - Unifying terminology for env/configuration. In this project, we call "configuration" to the set of environment
//      vars that Docker compose uses. There are some naming and concept clashes between the "config" package, the
//      Viper config at dotenv/loader.go, and the code at system/env.go
//  - At the moment, stdout is being used both for showing information to the user and for logs. This can be confusing
//      sometimes. Consider logging to a file or something similar.

func main() {
	// Load .env file from the current working directory
	// Variables are available for all Cobra commands
	dotenv.LoadDotEnv()

	if err := requireCommand("docker"); err != nil {
		exitWithCommandMissingError("docker")
	}
	if err := requireCommand("restic"); err != nil {
		exitWithCommandMissingError("restic")
	}
	if err := requireCommand("sh"); err != nil {
		exitWithCommandMissingError("sh")
	}
	if err := requireCommand("cp"); err != nil {
		// In this project we call the system's "cp" command to copy files or directories. This is just lazy
		// of me because I don't want to maintain the Go code required to do this. In the future we can
		// use a library or write custom code to handle this scenario
		exitWithCommandMissingError("cp")
	}
	if err := cmd.Execute(); err != nil {
		// Cobra already prints the error; ensure non-zero exit for failure cases
		slog.Error("Command execution failed", "error", err.Error())
		os.Exit(1)
	}
}

func requireCommand(command string) error {
	_, err := exec.LookPath(command)
	return err
}

func exitWithCommandMissingError(command string) {
	slog.Error("A command was not found. Is it installed in PATH?", "command", command)
	os.Exit(2)
}
