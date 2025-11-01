package system

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type System interface {
	// TODO IMPLEMENT
	RequireCommand(command string) error
	// Wraps os.ExecCommand
	ExecCommand(command string) error
	RequireFilesInWd(filenames ...string) error
}

type DefaultSystem struct {
	fs fs
}

func NewDefaultSystem() *DefaultSystem {
	return &DefaultSystem{
		fs: osFs{},
	}
}

// RequireFilesInWd requires that certain files exist in the
// current working directory
func (f *DefaultSystem) RequireFilesInWd(filenames ...string) error {
	if len(filenames) == 0 {
		return nil
	}

	cwd, err := f.fs.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	var missingFiles []string
	for _, file := range filenames {
		path := fmt.Sprintf("%s/%s", cwd, file)
		if _, err := f.fs.Stat(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				missingFiles = append(missingFiles, file)
			} else {
				// Some other error returned (permission denied, etc.)
				return fmt.Errorf("failed to check file %s: %w", path, err)
			}
		}
	}

	if len(missingFiles) > 0 {
		return fmt.Errorf("required filenames not found: %s", strings.Join(missingFiles, ", "))
	}

	return nil
}
