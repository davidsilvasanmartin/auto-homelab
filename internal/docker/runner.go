package docker

import (
	"log/slog"

	"github.com/davidsilvasanmartin/auto-homelab/internal/system"
)

// Runner is the interface for running docker commands
type Runner interface {
	ComposeStart(services []string) error
	ComposeStop(services []string) error
}

// SystemRunner implements the Docker Runner using system commands calls
type SystemRunner struct {
	commands system.Commands
	files    system.FilesHandler
}

// NewSystemRunner creates a new Docker SystemRunner
func NewSystemRunner() *SystemRunner {
	return &SystemRunner{
		commands: system.NewDefaultCommands(),
		files:    system.NewDefaultFilesHandler(),
	}
}

// ComposeStart starts services by using the commands's docker compose command
func (r *SystemRunner) ComposeStart(services []string) error {
	allArgs := append([]string{"up", "-d"}, services...)
	return r.executeComposeCommand(allArgs...)
}

func (r *SystemRunner) ComposeStop(services []string) error {
	allArgs := append([]string{"stop"}, services...)
	return r.executeComposeCommand(allArgs...)
}

func (r *SystemRunner) executeComposeCommand(args ...string) error {
	if err := r.files.RequireFilesInWd("docker-compose.yml", ".env"); err != nil {
		return err
	}

	allArgs := append([]string{"compose"}, args...)
	cmd := r.commands.ExecCommand("docker", allArgs...)

	slog.Debug("Executing docker compose command",
		"command", "docker",
		"args", args,
	)

	return cmd.Run()
}
