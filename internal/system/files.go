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

func (d *DefaultFilesHandler) EmptyDir(path string) error {
	slog.Debug("Emptying directory", "path", path)

	if !filepath.IsAbs(path) {
		return fmt.Errorf("path is not absolute: %q. Please use an absolute path", path)
	}

	if err := d.stdlib.RemoveAll(path); err != nil {
		return fmt.Errorf("error removing directory %q: %w", path, err)
	}

	if err := d.CreateDirIfNotExists(path); err != nil {
		return err
	}

	slog.Debug("Directory emptied successfully", "path", path)
	return nil
}

func (d *DefaultFilesHandler) CopyDir(srcPath string, dstPath string) error {
	slog.Debug("Copying directory", "sourcePath", srcPath, "outputPath", dstPath)
	cmd := d.stdlib.ExecCommand("cp", "-r", srcPath, dstPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to copy directory: %w", err)
	}
	slog.Debug("Successfully copied directory", "sourcePath", srcPath, "outputPath", dstPath)
	return nil
}
