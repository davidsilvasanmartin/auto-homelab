package backup

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// PrepareBackupDirectory creates an empty backup directory by removing all existing contents
func PrepareBackupDirectory(outputPath string, stdout io.Writer) error {
	// Create the directory if it doesn't exist
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory %s: %w", outputPath, err)
	}

	slog.Info("Preparing backup directory", "outputPath", outputPath)

	entries, err := os.ReadDir(outputPath)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", outputPath, err)
	}
	for _, entry := range entries {
		itemPath := filepath.Join(outputPath, entry.Name())
		if err := os.RemoveAll(itemPath); err != nil {
			return fmt.Errorf("error removing item %s from %s: %w", itemPath, outputPath, err)
		}
		if entry.IsDir() {
			slog.Info("Removed sub-directory", "itemPath", itemPath)
		} else {
			slog.Info("Removed file/symlink", "itemPath", itemPath)
		}
	}

	slog.Info("Successfully prepared backup directory", "outputPath", outputPath)

	return nil
}

// GetRequiredEnv gets a required environment variable or returns an error
func GetRequiredEnv(varName string) (string, error) {
	value, exists := os.LookupEnv(varName)
	if !exists {
		return "", fmt.Errorf("missing required environment variable: %s", varName)
	}
	return value, nil
}

// copyDir recursively copies a directory from src to dst
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate the relative path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		// Construct destination path
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			// Create directory
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Copy file
		return copyFile(path, dstPath, info.Mode())
	})
}

// copyFile copies a single file from src to dst
func copyFile(src, dst string, mode os.FileMode) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}
	defer sourceFile.Close()

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory for %s: %w", dst, err)
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dst, err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy file from %s to %s: %w", src, dst, err)
	}

	// Set permissions
	if err := os.Chmod(dst, mode); err != nil {
		return fmt.Errorf("failed to set permissions on %s: %w", dst, err)
	}

	return nil
}
