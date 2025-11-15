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
	// EnsureFilesInWD requires that the files exist in the current working directory, or errors if they don't
	EnsureFilesInWD(filenames ...string) error
	// EnsureDirExists requires that a directory exists, or throws an error if it doesn't
	EnsureDirExists(path string) error
	// EmptyDir empties a directory. The directory must exist
	EmptyDir(path string) error
	// CopyDir copies a directory, from srcPath into dstPath
	CopyDir(srcPath string, dstPath string) error
	// Getwd gets the current working directory
	Getwd() (dir string, err error)
	// WriteFile writes the content to a file
	WriteFile(path string, data []byte) error
	// GetAbsPath gets the absolute path from a relative (or absolute) path and cleans it
	GetAbsPath(path string) (string, error)
}

const (
	defaultDirPerms  os.FileMode = 0o755
	defaultFilePerms os.FileMode = 0o644
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
	ErrFailedToWriteFile    = errors.New("failed to write file")
	ErrFailedToGetAbsPath   = errors.New("failed to get abs path")
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
	// At this point it is guaranteed that the directory exists. For example, if the path was a file,
	// MkdirAll would have returned an error
	slog.Debug("Created directory", "path", path)
	return nil
}

func (d *DefaultFilesHandler) EnsureFilesInWD(filenames ...string) error {
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

// EnsureDirExists requires that a directory exists, or throws an error if it doesn't
func (d *DefaultFilesHandler) EnsureDirExists(path string) error {
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

func (d *DefaultFilesHandler) Getwd() (dir string, err error) {
	return d.stdlib.Getwd()
}

func (d *DefaultFilesHandler) WriteFile(path string, data []byte) error {
	if err := d.stdlib.WriteFile(path, data, defaultFilePerms); err != nil {
		return fmt.Errorf("%w %q: %w", ErrFailedToWriteFile, path, err)
	}
	return nil
}

func (d *DefaultFilesHandler) GetAbsPath(path string) (string, error) {
	absPath, err := d.stdlib.FilepathAbs(path)
	if err != nil {
		return "", fmt.Errorf("%w %q: %w", ErrFailedToGetAbsPath, path, err)
	}
	return absPath, nil
}
