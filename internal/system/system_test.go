package system

import (
	"errors"
	"os"
	"testing"
)

// Test that a command that exists returns no error
func TestDefaultSystem_RequireCommand_CommandExists(t *testing.T) {
	mock := &mockStdlib{
		execLookPath: func(file string) (string, error) {
			return "/usr/bin/docker", nil
		},
	}
	sys := &DefaultSystem{stdlib: mock}

	err := sys.RequireCommand("docker")
	if err != nil {
		t.Errorf("expected no error for existing command, got %v", err)
	}
}

// Test that a command that doesn't exist returns an error
func TestDefaultSystem_RequireCommand_CommandNotFound(t *testing.T) {
	lookPathErr := errors.New("executable file not found")
	mock := &mockStdlib{
		execLookPath: func(file string) (string, error) {
			return "", lookPathErr
		},
	}
	sys := &DefaultSystem{stdlib: mock}

	err := sys.RequireCommand("nonexistent-command")

	if err == nil {
		t.Fatal("expected error for missing command, got nil")
	}
	if !errors.Is(err, lookPathErr) {
		t.Errorf("expected error to wrap %v, got: %v", lookPathErr, err)
	}
}

// Test that an empty list returns no error
func TestDefaultSystem_RequireFilesInWd_EmptyList(t *testing.T) {
	mock := &mockStdlib{}
	sys := &DefaultSystem{stdlib: mock}

	err := sys.RequireFilesInWd()

	if err != nil {
		t.Errorf("expected no error for empty file list, got %v", err)
	}
}

// Test that when all files exist, it returns no error
func TestDefaultSystem_RequireFilesInWd_AllFilesExist(t *testing.T) {
	mock := &mockStdlib{
		getwd: func() (string, error) {
			return "/home/user/project", nil
		},
		stat: func(name string) (os.FileInfo, error) {
			// No error = File exists
			return nil, nil
		},
	}
	sys := &DefaultSystem{stdlib: mock}

	err := sys.RequireFilesInWd("file1.txt", "file2.go", "config.yml")

	if err != nil {
		t.Errorf("expected no error when all files exist, got: %v", err)
	}
}

// Test that a missing file returns an error
func TestDefaultSystem_RequireFilesInWd_SingleFileMissing(t *testing.T) {
	mock := &mockStdlib{
		getwd: func() (string, error) {
			return "/home/user/project", nil
		},
		stat: func(name string) (os.FileInfo, error) {
			if name == "/home/user/project/missing.txt" {
				return nil, os.ErrNotExist
			}
			return nil, nil
		},
	}
	sys := DefaultSystem{stdlib: mock}

	err := sys.RequireFilesInWd("existing.txt", "missing.txt")

	if err == nil {
		t.Fatal("expected error when file is missing, got nil")
	}
	expectedMsg := "required filenames not found: missing.txt"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
	}
}

// Test that multiple missing files are all reported
func TestDefaultSystem_RequireFilesInWd_MultipleFilesMissing(t *testing.T) {
	mock := &mockStdlib{
		getwd: func() (string, error) {
			return "/project", nil
		},
		stat: func(name string) (os.FileInfo, error) {
			if name == "/project/missing1.txt" || name == "/project/missing2.txt" {
				return nil, os.ErrNotExist
			}
			return nil, nil
		},
	}
	sys := &DefaultSystem{stdlib: mock}

	err := sys.RequireFilesInWd("file1.txt", "missing1.txt", "file2.txt", "missing2.txt")

	if err == nil {
		t.Fatal("expected error when files are missing, got nil")
	}
	expectedMsg := "required filenames not found: missing1.txt, missing2.txt"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
	}
}

// Test that an error getting the working directory is propagated
func TestDefaultSystem_RequireFilesInWd_GetwdError(t *testing.T) {
	expectedErr := errors.New("permission denied")
	mock := &mockStdlib{
		getwd: func() (string, error) {
			return "", expectedErr
		},
	}
	sys := &DefaultSystem{stdlib: mock}

	err := sys.RequireFilesInWd("file.txt")

	if err == nil {
		t.Fatal("expected error when Getwd fails, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got: %v", expectedErr, err)
	}
}

// Test that a non-ErrNotExist error from Stat is propagated
func TestDefaultSystem_RequireFilesInWd_StatError(t *testing.T) {
	expectedErr := errors.New("permission denied")
	mock := &mockStdlib{
		getwd: func() (string, error) {
			return "/project", nil
		},
		stat: func(name string) (os.FileInfo, error) {
			if name == "/project/restricted.txt" {
				return nil, expectedErr
			}
			return nil, nil
		},
	}
	sys := &DefaultSystem{stdlib: mock}

	err := sys.RequireFilesInWd("normal.txt", "restricted.txt")

	if err == nil {
		t.Fatal("expected error when Stat fails, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got: %v", expectedErr, err)
	}
}

// Test that paths are constructed correctly
func TestDefaultSystem_RequireFilesInWd_CorrectPathConstruction(t *testing.T) {
	var checkedPaths []string
	mock := &mockStdlib{
		getwd: func() (string, error) {
			return "/home/user/project", nil
		},
		stat: func(name string) (os.FileInfo, error) {
			checkedPaths = append(checkedPaths, name)
			return nil, nil
		},
	}
	sys := &DefaultSystem{stdlib: mock}

	_ = sys.RequireFilesInWd("f1.go", "subdir/f2.go", "/subdirWithSlash/f3.go")
	expectedPaths := []string{
		"/home/user/project/f1.go",
		"/home/user/project/subdir/f2.go",
		// TODO fix double slash
		"/home/user/project//subdirWithSlash/f3.go",
	}

	if len(checkedPaths) != len(expectedPaths) {
		t.Fatalf("expected %d paths to be checked, got %d", len(expectedPaths), len(checkedPaths))
	}
	for i, expected := range expectedPaths {
		if checkedPaths[i] != expected {
			t.Errorf("path %d: expected %q, got %q", i, expected, checkedPaths[i])
		}
	}
}
