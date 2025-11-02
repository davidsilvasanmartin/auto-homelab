package system

import (
	"errors"
	"os"
	"testing"
)

func TestDefaultSystem_RequireFilesInWd_EmptyList(t *testing.T) {
	mock := &mockStdlib{}
	files := &DefaultFilesHandler{stdlib: mock}

	err := files.RequireFilesInWd()

	if err != nil {
		t.Errorf("expected no error for empty file list, got %v", err)
	}
}

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
	files := &DefaultFilesHandler{stdlib: mock}

	err := files.RequireFilesInWd("file1.txt", "file2.go", "config.yml")

	if err != nil {
		t.Errorf("expected no error when all files exist, got: %v", err)
	}
}

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
	files := &DefaultFilesHandler{stdlib: mock}

	err := files.RequireFilesInWd("existing.txt", "missing.txt")

	if err == nil {
		t.Fatal("expected error when file is missing, got nil")
	}
	expectedMsg := "required filenames not found: missing.txt"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
	}
}

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
	files := &DefaultFilesHandler{stdlib: mock}

	err := files.RequireFilesInWd("file1.txt", "missing1.txt", "file2.txt", "missing2.txt")

	if err == nil {
		t.Fatal("expected error when files are missing, got nil")
	}
	expectedMsg := "required filenames not found: missing1.txt, missing2.txt"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestDefaultSystem_RequireFilesInWd_GetwdError(t *testing.T) {
	expectedErr := errors.New("permission denied")
	mock := &mockStdlib{
		getwd: func() (string, error) {
			return "", expectedErr
		},
	}
	files := &DefaultFilesHandler{stdlib: mock}

	err := files.RequireFilesInWd("file.txt")

	if err == nil {
		t.Fatal("expected error when Getwd fails, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got: %v", expectedErr, err)
	}
}

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
	files := &DefaultFilesHandler{stdlib: mock}

	err := files.RequireFilesInWd("normal.txt", "restricted.txt")

	if err == nil {
		t.Fatal("expected error when Stat fails, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got: %v", expectedErr, err)
	}
}

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
	files := &DefaultFilesHandler{stdlib: mock}

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

func TestDefaultSystem_RequireDir_NotFound(t *testing.T) {
	mock := &mockStdlib{
		stat: func(name string) (os.FileInfo, error) {
			return nil, os.ErrNotExist
		},
	}
	system := &DefaultFilesHandler{stdlib: mock}

	err := system.RequireDir("/home/user/dir")

	if err == nil {
		t.Fatal("expected error when dir missing, got nil")
	}
	expectedMsg := "required directory not found: /home/user/dir"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestDefaultSystem_RequireDir_GenericError(t *testing.T) {
	mock := &mockStdlib{
		stat: func(name string) (os.FileInfo, error) {
			return nil, errors.New("permission denied")
		},
	}
	system := &DefaultFilesHandler{stdlib: mock}

	err := system.RequireDir("/home/user/dir")

	if err == nil {
		t.Fatal("expected error when dir cannot be checked, got nil")
	}
	expectedMsg := "failed to check file /home/user/dir: permission denied"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestDefaultSystem_RequireDir_NotADir(t *testing.T) {
	mock := &mockStdlib{
		stat: func(name string) (os.FileInfo, error) {
			mockFile := &mockFileInfo{
				name:  "file.txt",
				isDir: false,
			}
			return mockFile, nil
		},
	}
	system := &DefaultFilesHandler{stdlib: mock}

	err := system.RequireDir("/home/user/file.txt")

	if err == nil {
		t.Fatal("expected error when dir is a file, got nil")
	}
	expectedMsg := "source path /home/user/file.txt is not a directory"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestDefaultSystem_RequireDir_Ok(t *testing.T) {
	mock := &mockStdlib{
		stat: func(name string) (os.FileInfo, error) {
			mockFile := &mockFileInfo{
				name:  "dir",
				isDir: true,
			}
			return mockFile, nil
		},
	}
	system := &DefaultFilesHandler{stdlib: mock}

	err := system.RequireDir("/home/user/dir")

	if err != nil {
		t.Errorf("expected no error when directory exists, got %v", err)
	}
}
