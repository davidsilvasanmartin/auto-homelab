package system

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

type FilesHandler interface {
	// CreateDirIfNotExists creates the directory at the specified path if it doesn't exist
	CreateDirIfNotExists(path string) error
	// RequireFilesInWd requires that the files exist in the current working directory, or errors if they don't
	RequireFilesInWd(filenames ...string) error
	// RequireDir requires that a directory exists
	RequireDir(path string) error
	// EmptyDir empties a directory. The directory must exist
	EmptyDir(path string) error
	// CopyDir copies a directory, from srcPath into dstPath
	CopyDir(srcPath string, dstPath string) error
}

const (
	defaultDirPerms os.FileMode = 0o755
)

type DefaultFilesHandler struct {
	stdlib stdlib
}

func NewDefaultFilesHandler() *DefaultFilesHandler {
	return &DefaultFilesHandler{
		stdlib: newGoStdlib(),
	}
}

func (d *DefaultFilesHandler) CreateDirIfNotExists(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path is not absolute: %q. Please use an absolute path", path)
	}
	cleanPath := filepath.Clean(path)
	if err := d.stdlib.MkdirAll(cleanPath, defaultDirPerms); err != nil {
		return fmt.Errorf("failed to create directory %q: %w", cleanPath, err)
	}
	slog.Debug("Created directory", "path", path)
	return nil
}

func (d *DefaultFilesHandler) RequireFilesInWd(filenames ...string) error {
	if len(filenames) == 0 {
		return nil
	}

	cwd, err := d.stdlib.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	var missingFiles []string
	for _, file := range filenames {
		path := filepath.Join(cwd, file)
		if _, err := d.stdlib.Stat(path); err != nil {
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

// RequireDir requires that a directory exists, or throws an error if it doesn't
func (d *DefaultFilesHandler) RequireDir(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path is not absolute: %q. Please use an absolute path", path)
	}
	if stat, err := d.stdlib.Stat(path); err != nil && errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("required directory not found: %s", path)
	} else if err != nil {
		return fmt.Errorf("failed to check file %s: %w", path, err)
	} else if !stat.IsDir() {
		return fmt.Errorf("source path %s is not a directory", path)
	}
	return nil
}

// EmptyDir empties a directory if it exists. If the directory does not exist, this method will create it
func (d *DefaultFilesHandler) EmptyDir(path string) error {
	slog.Debug("Emptying directory", "path", path)

	if !filepath.IsAbs(path) {
		return fmt.Errorf("path is not absolute: %q. Please use an absolute path", path)
	}

	// TODO clean the path, and TEST (see `TestDefaultFilesHandler_CreateDirIfNotExists_CleansDirtyPath`)
	if err := d.stdlib.RemoveAll(path); err != nil {
		return fmt.Errorf("error removing directory %q: %w", path, err)
	}

	if err := d.CreateDirIfNotExists(path); err != nil {
		return err
	}

	slog.Debug("Directory emptied successfully", "path", path)
	return nil
}

// CopyDir copies a directory by using the system's cp command. This is not portable: if we want the project to work
// on Windows, we will have to change this
func (d *DefaultFilesHandler) CopyDir(srcPath string, dstPath string) error {
	// TODO think about requiring absolute paths
	slog.Debug("Copying directory", "srcPath", srcPath, "dstPath", dstPath)
	cmd := d.stdlib.ExecCommand("cp", "-r", srcPath, dstPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to copy directory: %w", err)
	}
	slog.Debug("Successfully copied directory", "srcPath", srcPath, "dstPath", dstPath)
	return nil
}
