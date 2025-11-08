package system

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDefaultFilesHandler_CreateDirIfNotExists_Success(t *testing.T) {
	var createdPath string
	var createdPerm os.FileMode
	std := &mockStdlib{
		mkdirAll: func(path string, mode os.FileMode) error {
			createdPath = path
			createdPerm = mode
			return nil
		},
	}
	files := &DefaultFilesHandler{stdlib: std}

	err := files.CreateDirIfNotExists("/home/user/newdir")

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if createdPath != "/home/user/newdir" {
		t.Errorf("expected path %q, got %q", "/home/user/newdir", createdPath)
	}
	if createdPerm != defaultDirPerms {
		t.Errorf("expected permissions %o, got %o", defaultDirPerms, createdPerm)
	}
}

func TestDefaultFilesHandler_CreateDirIfNotExists_PathNotAbs(t *testing.T) {
	files := &DefaultFilesHandler{stdlib: &mockStdlib{}}

	err := files.CreateDirIfNotExists("relative/path")

	if err == nil {
		t.Fatal("expected error when path is relative, got nil")
	}
	if !errors.Is(err, ErrPathNotAbsolute) {
		t.Errorf("expected ErrPathNotAbsolute, got: %v", err)
	}
}

func TestDefaultFilesHandler_CreateDirIfNotExists_CleansDirtyPath(t *testing.T) {
	var createdPath string
	std := &mockStdlib{
		mkdirAll: func(path string, mode os.FileMode) error {
			createdPath = path
			return nil
		},
	}
	files := &DefaultFilesHandler{stdlib: std}

	err := files.CreateDirIfNotExists("/home/user//./subdir/../newdir/")

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	expectedPath := "/home/user/newdir"
	if createdPath != expectedPath {
		t.Errorf("expected cleaned path %q, got %q", expectedPath, createdPath)
	}
}

func TestDefaultFilesHandler_CreateDirIfNotExists_MkdirAllError(t *testing.T) {
	expectedErr := errors.New("permission denied")
	std := &mockStdlib{
		mkdirAll: func(path string, mode os.FileMode) error {
			return expectedErr
		},
	}
	files := &DefaultFilesHandler{stdlib: std}

	err := files.CreateDirIfNotExists("/home/user/restricteddir")

	if err == nil {
		t.Fatal("expected error when MkdirAll fails, got nil")
	}
	if !errors.Is(err, ErrFailedToCreateDir) {
		t.Errorf("expected ErrFailedToCreateDir, got: %v", err)
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got: %v", expectedErr, err)
	}
}

func TestDefaultFilesHandler_RequireFilesInWd_EmptyList(t *testing.T) {
	std := &mockStdlib{}
	files := &DefaultFilesHandler{stdlib: std}

	err := files.RequireFilesInWd()

	if err != nil {
		t.Errorf("expected no error for empty file list, got %v", err)
	}
}

func TestDefaultFilesHandler_RequireFilesInWd_AllFilesExist(t *testing.T) {
	std := &mockStdlib{
		getwd: func() (string, error) {
			return "/home/user/project", nil
		},
		stat: func(name string) (os.FileInfo, error) {
			// No error = File exists
			return nil, nil
		},
	}
	files := &DefaultFilesHandler{stdlib: std}

	err := files.RequireFilesInWd("file1.txt", "file2.go", "config.yml")

	if err != nil {
		t.Errorf("expected no error when all files exist, got: %v", err)
	}
}

func TestDefaultFilesHandler_RequireFilesInWd_SingleFileMissing(t *testing.T) {
	std := &mockStdlib{
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
	files := &DefaultFilesHandler{stdlib: std}

	err := files.RequireFilesInWd("existing.txt", "missing.txt")

	if err == nil {
		t.Fatal("expected error when file is missing, got nil")
	}
	if !errors.Is(err, ErrRequiredFileNotFound) {
		t.Errorf("expected ErrRequiredFileNotFound, got: %v", err)
	}
	if !strings.Contains(err.Error(), "missing.txt") {
		t.Errorf("expected error message to contain %q, got %q", "missing.txt", err.Error())
	}
}

func TestDefaultFilesHandler_RequireFilesInWd_MultipleFilesMissing(t *testing.T) {
	std := &mockStdlib{
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
	files := &DefaultFilesHandler{stdlib: std}

	err := files.RequireFilesInWd("file1.txt", "missing1.txt", "file2.txt", "missing2.txt")

	if err == nil {
		t.Fatal("expected error when files are missing, got nil")
	}
	if !errors.Is(err, ErrRequiredFileNotFound) {
		t.Errorf("expected ErrRequiredFileNotFound, got: %v", err)
	}
	if !strings.Contains(err.Error(), "missing1.txt") {
		t.Errorf("expected error message to contain %q, got %q", "missing1.txt", err.Error())
	}
	if !strings.Contains(err.Error(), "missing2.txt") {
		t.Errorf("expected error message to contain %q, got %q", "missing2.txt", err.Error())
	}
}

func TestDefaultFilesHandler_RequireFilesInWd_GetwdError(t *testing.T) {
	expectedErr := errors.New("permission denied")
	std := &mockStdlib{
		getwd: func() (string, error) {
			return "", expectedErr
		},
	}
	files := &DefaultFilesHandler{stdlib: std}

	err := files.RequireFilesInWd("file.txt")

	if err == nil {
		t.Fatal("expected error when Getwd fails, got nil")
	}
	if !errors.Is(err, ErrFailedToCheckPath) {
		t.Errorf("expected ErrFailedToCheckPath, got: %v", err)
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got: %v", expectedErr, err)
	}
}

func TestDefaultFilesHandler_RequireFilesInWd_StatError(t *testing.T) {
	expectedErr := errors.New("permission denied")
	std := &mockStdlib{
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
	files := &DefaultFilesHandler{stdlib: std}

	err := files.RequireFilesInWd("normal.txt", "restricted.txt")

	if err == nil {
		t.Fatal("expected error when Stat fails, got nil")
	}
	if !errors.Is(err, ErrFailedToCheckPath) {
		t.Errorf("expected ErrFailedToCheckPath, got: %v", err)
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got: %v", expectedErr, err)
	}
}

func TestDefaultFilesHandler_RequireFilesInWd_CorrectPathConstruction(t *testing.T) {
	var checkedPaths []string
	std := &mockStdlib{
		getwd: func() (string, error) {
			return "/home/user/project", nil
		},
		stat: func(name string) (os.FileInfo, error) {
			checkedPaths = append(checkedPaths, name)
			return nil, nil
		},
	}
	files := &DefaultFilesHandler{stdlib: std}

	_ = files.RequireFilesInWd("f1.go", "subdir/f2.go", "/../project///subdirWithSlash/f3.go")
	expectedPaths := []string{
		"/home/user/project/f1.go",
		"/home/user/project/subdir/f2.go",
		"/home/user/project/subdirWithSlash/f3.go",
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

func TestDefaultFilesHandler_RequireDir_PathNotAbs(t *testing.T) {
	files := &DefaultFilesHandler{stdlib: &mockStdlib{}}

	err := files.RequireDir("./dir/subdir")

	if err == nil {
		t.Fatal("expected error when path is relative, got nil")
	}
	if !errors.Is(err, ErrPathNotAbsolute) {
		t.Errorf("expected ErrPathNotAbsolute, got: %v", err)
	}
}

func TestDefaultFilesHandler_RequireDir_NotFound(t *testing.T) {
	std := &mockStdlib{
		stat: func(name string) (os.FileInfo, error) {
			return nil, os.ErrNotExist
		},
	}
	files := &DefaultFilesHandler{stdlib: std}
	path := "/home/user/dir"

	err := files.RequireDir(path)

	if err == nil {
		t.Fatal("expected error when dir missing, got nil")
	}
	if !errors.Is(err, ErrRequiredDirNotFound) {
		t.Errorf("expected ErrRequiredDirNotFound, got: %v", err)
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("expected error message to contain path %q, got %q", path, err.Error())
	}
}

func TestDefaultFilesHandler_RequireDir_GenericError(t *testing.T) {
	std := &mockStdlib{
		stat: func(name string) (os.FileInfo, error) {
			return nil, errors.New("permission denied")
		},
	}
	files := &DefaultFilesHandler{stdlib: std}
	path := "/home/user/dir"

	err := files.RequireDir(path)

	if err == nil {
		t.Fatal("expected error when dir cannot be checked, got nil")
	}
	if !errors.Is(err, ErrFailedToCheckPath) {
		t.Errorf("expected ErrFailedToCheckPath, got: %v", err)
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("expected error message to contain path %q, got %q", path, err.Error())
	}
}

func TestDefaultFilesHandler_RequireDir_NotADir(t *testing.T) {
	std := &mockStdlib{
		stat: func(name string) (os.FileInfo, error) {
			mockFile := &mockFileInfo{
				name:  "file.txt",
				isDir: false,
			}
			return mockFile, nil
		},
	}
	files := &DefaultFilesHandler{stdlib: std}

	err := files.RequireDir("/home/user/file.txt")

	if err == nil {
		t.Fatal("expected error when dir is a file, got nil")
	}
	if !errors.Is(err, ErrNotADir) {
		t.Errorf("expected ErrNotADir, got: %v", err)
	}
	if !strings.Contains(err.Error(), "/home/user/file.txt") {
		t.Errorf("expected error message to contain path %q, got %q", "/home/user/file.txt", err.Error())
	}
}

func TestDefaultFilesHandler_RequireDir_Success(t *testing.T) {
	std := &mockStdlib{
		stat: func(name string) (os.FileInfo, error) {
			mockFile := &mockFileInfo{
				name:  "dir",
				isDir: true,
			}
			return mockFile, nil
		},
	}
	files := &DefaultFilesHandler{stdlib: std}

	err := files.RequireDir("/home/user/dir")

	if err != nil {
		t.Errorf("expected no error when directory exists, got %v", err)
	}
}

func TestDefaultFilesHandler_EmptyDir_Success(t *testing.T) {
	var removedPath string
	var createdPath string
	var createdPerm os.FileMode
	std := &mockStdlib{
		removeAll: func(path string) error {
			removedPath = path
			return nil
		},
		mkdirAll: func(path string, mode os.FileMode) error {
			createdPath = path
			createdPerm = mode
			return nil
		},
	}
	files := &DefaultFilesHandler{stdlib: std}

	err := files.EmptyDir("/home/user/targetdir")

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if removedPath != "/home/user/targetdir" {
		t.Errorf("expected removed path %q, got %q", "/home/user/targetdir", removedPath)
	}
	if createdPath != "/home/user/targetdir" {
		t.Errorf("expected created path %q, got %q", "/home/user/targetdir", createdPath)
	}
	if createdPerm != defaultDirPerms {
		t.Errorf("expected permissions %o, got %o", defaultDirPerms, createdPerm)
	}
}

func TestDefaultFilesHandler_EmptyDir_CleansPath(t *testing.T) {
	var removedPath string
	var createdPath string
	std := &mockStdlib{
		removeAll: func(path string) error {
			removedPath = path
			return nil
		},
		mkdirAll: func(path string, mode os.FileMode) error {
			createdPath = path
			return nil
		},
	}
	files := &DefaultFilesHandler{stdlib: std}

	err := files.EmptyDir("/home//./user/../user///targetdir")

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if removedPath != "/home/user/targetdir" {
		t.Errorf("expected clean removed path %q, got %q", "/home/user/targetdir", removedPath)
	}
	if createdPath != "/home/user/targetdir" {
		t.Errorf("expected clean created path %q, got %q", "/home/user/targetdir", createdPath)
	}
}

func TestDefaultFilesHandler_EmptyDir_PathNotAbs(t *testing.T) {
	files := &DefaultFilesHandler{stdlib: &mockStdlib{}}

	err := files.EmptyDir("relative/path")

	if err == nil {
		t.Fatal("expected error when path is relative, got nil")
	}
	if !errors.Is(err, ErrPathNotAbsolute) {
		t.Errorf("expected ErrPathNotAbsolute, got: %v", err)
	}
}

func TestDefaultFilesHandler_EmptyDir_RemoveAllError(t *testing.T) {
	expectedErr := errors.New("disk full")
	std := &mockStdlib{
		removeAll: func(path string) error {
			return expectedErr
		},
	}
	files := &DefaultFilesHandler{stdlib: std}
	path := "/home/user/somedir"

	err := files.EmptyDir(path)

	if err == nil {
		t.Fatal("expected error when RemoveAll fails, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got: %v", expectedErr, err)
	}
	if !errors.Is(err, ErrFailedToRemoveDir) {
		t.Errorf("expected ErrFailedToRemoveDir, got: %v", err)
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("expected error message to contain path %q, got %q", path, err.Error())
	}
}

func TestDefaultFilesHandler_EmptyDir_MkdirAllError(t *testing.T) {
	expectedErr := errors.New("permission denied")
	std := &mockStdlib{
		removeAll: func(path string) error {
			return nil
		},
		mkdirAll: func(path string, mode os.FileMode) error {
			return expectedErr
		},
	}
	files := &DefaultFilesHandler{stdlib: std}
	path := "/home/user/restricteddir"

	err := files.EmptyDir(path)

	if err == nil {
		t.Fatal("expected error when MkdirAll fails, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got: %v", expectedErr, err)
	}
	if !errors.Is(err, ErrFailedToCreateDir) {
		t.Errorf("expected ErrFailedToCreateDir, got: %v", err)
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("expected error message to contain path %q, got %q", path, err.Error())
	}
}

func TestDefaultFilesHandler_CopyDir_Success(t *testing.T) {
	var commandName string
	var commandArgs []string
	mockCmd := &mockRunnableCommand{
		runFunc: func() error {
			return nil
		},
	}
	std := &mockStdlib{
		execCommand: func(name string, arg ...string) RunnableCommand {
			commandName = name
			commandArgs = arg
			return mockCmd
		},
	}
	files := &DefaultFilesHandler{stdlib: std}

	err := files.CopyDir("/home/user/src", "/home/user/dst")

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if commandName != "cp" {
		t.Errorf("expected command name %q, got %q", "cp", commandName)
	}
	expectedArgs := []string{"-r", "/home/user/src", "/home/user/dst"}
	if diff := cmp.Diff(expectedArgs, commandArgs); diff != "" {
		t.Errorf("args mismatch (-want +got):\n%s", diff)
	}
}

func TestDefaultFilesHandler_CopyDir_CleansPaths(t *testing.T) {
	var commandArgs []string
	mockCmd := &mockRunnableCommand{
		runFunc: func() error {
			return nil
		},
	}
	std := &mockStdlib{
		execCommand: func(name string, arg ...string) RunnableCommand {
			commandArgs = arg
			return mockCmd
		},
	}
	files := &DefaultFilesHandler{stdlib: std}

	err := files.CopyDir("/home/./user/../user////src", "/home/./user/../user///dst")

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	expectedArgs := []string{"-r", "/home/user/src", "/home/user/dst"}
	if diff := cmp.Diff(expectedArgs, commandArgs); diff != "" {
		t.Errorf("args mismatch (-want +got):\n%s", diff)
	}
}

func TestDefaultFilesHandler_CopyDir_CommandError(t *testing.T) {
	expectedErr := errors.New("source directory not found")
	mockCmd := &mockRunnableCommand{
		runFunc: func() error {
			return expectedErr
		},
	}
	std := &mockStdlib{
		execCommand: func(name string, arg ...string) RunnableCommand {
			return mockCmd
		},
	}
	files := &DefaultFilesHandler{stdlib: std}
	srcPath := "/home/user/src"
	dstPath := "/home/user/dst"

	err := files.CopyDir(srcPath, dstPath)

	if err == nil {
		t.Fatal("expected error when cp command fails, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got: %v", expectedErr, err)
	}
	if !errors.Is(err, ErrFailedToCopyDir) {
		t.Errorf("expected ErrFailedToCopyDir, got: %v", err)
	}
	if !strings.Contains(err.Error(), srcPath) {
		t.Errorf("expected error message to contain path %q, got %q", srcPath, err.Error())
	}
	if !strings.Contains(err.Error(), dstPath) {
		t.Errorf("expected error message to contain path %q, got %q", dstPath, err.Error())
	}
}
