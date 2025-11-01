package docker

import (
	"github.com/davidsilvasanmartin/auto-homelab/internal/system"
	"io"
	"log/slog"
)

// Runner is the interface for running docker commands
type Runner interface {
	ComposeStart(services []string) error
}

// SystemRunner implements the Docker Runner using system calls
type SystemRunner struct {
	stdout io.Writer
	stderr io.Writer
	system system.System
}

// NewSystemRunner creates a new SystemRunner with stdout and stderr
func NewSystemRunner(stdout io.Writer, stderr io.Writer) *SystemRunner {
	return &SystemRunner{
		stdout: stdout,
		stderr: stderr,
		system: system.NewDefaultSystem(),
	}
}

// ComposeStart starts services by using the system's docker compose command
func (r *SystemRunner) ComposeStart(services []string) error {
	if err := r.system.RequireCommand("docker"); err != nil {
		return err
	}
	if err := r.system.RequireFilesInWd("docker-compose.yml", ".env"); err != nil {
		return err
	}

	args := []string{"compose", "up", "-d"}
	args = append(args, services...)

	cmd := r.system.ExecCommand("docker", args...)
	cmd.Stdout = r.stdout
	cmd.Stderr = r.stderr
	cmd.Dir = "."

	slog.Debug("Executing docker compose command",
		"command", "docker",
		"args", args,
		"dir", cmd.Dir,
	)

	return cmd.Run()
}
