package require

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Require interface {
	RequireDocker() error
	RequireFilesInWd(files ...string) error
}

// SystemRequire utilizes system utilities to perform the require operations
type SystemRequire struct {
}

// TODO "New" constructor

// RequireDocker requires that the docker command is installed,
// or returns an error otherwise
func (r *SystemRequire) RequireDocker() error {
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("docker command not found: %w (is Docker installed in PATH?)", err)
	}
	return nil
}

// RequireFilesInWd requires that certain files exist in the
// current working directory
func (r *SystemRequire) RequireFilesInWd(files ...string) error {
	// TODO
	if len(files) == 0 {
		return nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	var missingFiles []string
	for _, file := range files {
		path := fmt.Sprintf("%s/%s", cwd, file)
		if _, err := os.Stat(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				missingFiles = append(missingFiles, file)
			} else {
				// Some other error returned (permission denied, etc.)
				return fmt.Errorf("failed to check file %s: %w", path, err)
			}
		}
	}

	if len(missingFiles) > 0 {
		return fmt.Errorf("required files not found: %s", strings.Join(missingFiles, ", "))
	}

	return nil
}
