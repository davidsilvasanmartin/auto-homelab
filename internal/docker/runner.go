package docker

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/davidsilvasanmartin/auto-homelab/internal/system"
)

// Runner is the interface for running docker commands
type Runner interface {
	ComposeStart(services []string) error
	ComposeStop(services []string) error
	ContainerExec(container string, cmd string) error
	WaitUntilContainerExecIsSuccessful(container string, cmd string) error
}

var (
	ErrTooManyRetries = errors.New("too many retries")
)

// SystemRunner implements the Docker Runner using system commands calls
type SystemRunner struct {
	commands system.Commands
	files    system.FilesHandler
	time     system.Time
}

// NewSystemRunner creates a new Docker SystemRunner
func NewSystemRunner() *SystemRunner {
	return &SystemRunner{
		commands: system.NewDefaultCommands(),
		files:    system.NewDefaultFilesHandler(),
		time:     system.NewDefaultTime(),
	}
}

// ComposeStart starts services by using the system's docker compose command
func (r *SystemRunner) ComposeStart(services []string) error {
	allArgs := append([]string{"up", "-d"}, services...)
	return r.executeComposeCommand(allArgs...)
}

// ComposeStop stops services by using the system's docker compose command
func (r *SystemRunner) ComposeStop(services []string) error {
	allArgs := append([]string{"stop"}, services...)
	return r.executeComposeCommand(allArgs...)
}

func (r *SystemRunner) executeComposeCommand(args ...string) error {
	// The docker compose command will automatically read the .env file,
	if err := r.files.EnsureFilesInWD("docker-compose.yml", ".env"); err != nil {
		return err
	}

	// Set UID and GID environment variables for docker compose .These are used by the user: directive in
	// docker-compose.yml. Note that these variables are NOT needed for "docker compose exec" commands because
	// those commands are executed on already running containers.
	uid := os.Getuid()
	gid := os.Getgid()

	var cmdParts []string
	cmdParts = append(cmdParts, fmt.Sprintf("HOMELAB_GENERAL_UID=%d", uid))
	cmdParts = append(cmdParts, fmt.Sprintf("HOMELAB_GENERAL_GID=%d", gid))
	cmdParts = append(cmdParts, "docker compose")
	cmdParts = append(cmdParts, args...)

	fullCmd := strings.Join(cmdParts, " ")
	cmd := r.commands.ExecShellCommand(fullCmd)

	return cmd.Run()
}

func (r *SystemRunner) ContainerExec(container string, cmd string) error {
	fullCmd := fmt.Sprintf("docker container exec %s %s", container, cmd)
	systemCmd := r.commands.ExecShellCommand(fullCmd)
	return systemCmd.Run()
}

func (r *SystemRunner) WaitUntilContainerExecIsSuccessful(container string, cmd string) error {
	slog.Debug("Waiting until docker container exec command is successful", "container", container, "cmd", cmd)
	maxRetries := 30
	retryInterval := 1 * time.Second

	for i := 0; i < maxRetries; i++ {
		if err := r.ContainerExec(container, cmd); err == nil {
			return nil
		}
		r.time.Sleep(retryInterval)
	}

	return fmt.Errorf("%w: docker container exec command %q retried %d times on container %s", ErrTooManyRetries, cmd, maxRetries, container)
}
