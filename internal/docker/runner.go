package docker

import (
	"auto-homelab/internal/system"
	"io"
	"log/slog"
	"os/exec"
)

// DockerRunner is the interface for running docker commands
type DockerRunner interface {
	ComposeStart(services []string) error
}

// SystemDockerRunner implements DockerRunner using actual system calls
type SystemDockerRunner struct {
	stdout      io.Writer
	stderr      io.Writer
	execCommand func(name string, arg ...string) *exec.Cmd
	files       system.System
}

// NewSystemDockerRunner creates a new SystemDockerRunner with stdout and stderr
func NewSystemDockerRunner(stdout io.Writer, stderr io.Writer) *SystemDockerRunner {
	return &SystemDockerRunner{
		stdout:      stdout,
		stderr:      stderr,
		execCommand: exec.Command,
		files:       system.NewDefaultSystem(),
	}
}

// ComposeStart starts services by using the system's docker compose command
func (r *SystemDockerRunner) ComposeStart(services []string) error {
	if err := r.require.RequireDocker(); err != nil {
		return err
	}
	if err := r.files.RequireFilesInWd("docker-compose.yml", ".env"); err != nil {
		return err
	}

	args := []string{"compose", "up", "-d"}
	args = append(args, services...)

	cmd := r.execCommand("docker", args...)
	cmd.Stdout = r.stdout
	cmd.Stderr = r.stderr

	slog.Debug("Executing docker compose command",
		"command", "docker",
		"args", args,
		"dir", cmd.Dir,
	)

	return cmd.Run()
}
