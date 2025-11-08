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

var (
	ErrPathNotAbsolute      = errors.New("path is not absolute")
	ErrRequiredFileNotFound = errors.New("required file not found")
	ErrRequiredDirNotFound  = errors.New("required directory not found")
	ErrNotADir              = errors.New("path is not a directory")
	ErrFailedToCreateDir    = errors.New("failed to create directory")
	ErrFailedToRemoveDir    = errors.New("failed to remove directory")
	ErrFailedToCopyDir      = errors.New("failed to copy directory")
	ErrFailedToCheckPath    = errors.New("failed to check file or directory at path")
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
		return fmt.Errorf("%w: %q", ErrPathNotAbsolute, path)
	}
	cleanPath := filepath.Clean(path)
	if err := d.stdlib.MkdirAll(cleanPath, defaultDirPerms); err != nil {
		return fmt.Errorf("%w %q: %w", ErrFailedToCreateDir, cleanPath, err)
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
		return fmt.Errorf("%w %q: %w", ErrFailedToCheckPath, ".", err)
	}

	var missingFiles []string
	for _, file := range filenames {
		path := filepath.Join(cwd, file)
		if _, err := d.stdlib.Stat(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				missingFiles = append(missingFiles, file)
			} else {
				// Some other error returned (permission denied, etc.)
				return fmt.Errorf("%w %q: %w", ErrFailedToCheckPath, path, err)
			}
		}
	}

	if len(missingFiles) > 0 {
		return fmt.Errorf("%w: %s", ErrRequiredFileNotFound, strings.Join(missingFiles, ", "))
	}

	return nil
}

// RequireDir requires that a directory exists, or throws an error if it doesn't
func (d *DefaultFilesHandler) RequireDir(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("%w: %q", ErrPathNotAbsolute, path)
	}
	if stat, err := d.stdlib.Stat(path); err != nil && errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w: %s", ErrRequiredDirNotFound, path)
	} else if err != nil {
		return fmt.Errorf("%w %q: %w", ErrFailedToCheckPath, path, err)
	} else if !stat.IsDir() {
		return fmt.Errorf("%w: %q", ErrNotADir, path)
	}
	return nil
}

// EmptyDir empties a directory if it exists. If the directory does not exist, this method will create it
func (d *DefaultFilesHandler) EmptyDir(path string) error {
	cleanPath := filepath.Clean(path)
	slog.Debug("Emptying directory", "path", cleanPath)

	if !filepath.IsAbs(cleanPath) {
		return fmt.Errorf("%w: %q", ErrPathNotAbsolute, cleanPath)
	}

	if err := d.stdlib.RemoveAll(cleanPath); err != nil {
		return fmt.Errorf("%w %q: %w", ErrFailedToRemoveDir, cleanPath, err)
	}

	if err := d.CreateDirIfNotExists(cleanPath); err != nil {
		return err
	}

	slog.Debug("Directory emptied successfully", "path", cleanPath)
	return nil
}

// CopyDir copies a directory by using the system's cp command. Accepts absolute or relative paths.
// Using cp is not portable: if we want the project to work on Windows, we will have to change this
func (d *DefaultFilesHandler) CopyDir(srcPath string, dstPath string) error {
	cleanSrcPath := filepath.Clean(srcPath)
	cleanDstPath := filepath.Clean(dstPath)
	slog.Debug("Copying directory", "srcPath", cleanSrcPath, "dstPath", cleanDstPath)
	cmd := d.stdlib.ExecCommand("cp", "-r", cleanSrcPath, cleanDstPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w (%q to %q): %w", ErrFailedToCopyDir, cleanSrcPath, cleanDstPath, err)
	}
	slog.Debug("Successfully copied directory", "srcPath", cleanSrcPath, "dstPath", cleanDstPath)
	return nil
}
