Looking at your tests, I can see you're comparing error messages using string comparisons. In Go, it's indeed a best practice to use **sentinel errors** (predefined error variables) or **custom error types** for this purpose. This makes your code more maintainable and type-safe.

Let me show you how to refactor your code using both approaches:

## 1. Sentinel Errors (Recommended for Simple Cases)

This is the most common approach in Go. You define error variables that can be compared using `errors.Is()`.

```go
package system

import "errors"

// Sentinel errors for the system package
var (
	ErrPathNotAbsolute      = errors.New("path is not absolute")
	ErrRequiredFileNotFound = errors.New("required file not found")
	ErrRequiredDirNotFound  = errors.New("required directory not found")
	ErrNotADirectory        = errors.New("path is not a directory")
	ErrFailedToCreateDir    = errors.New("failed to create directory")
	ErrFailedToRemoveDir    = errors.New("failed to remove directory")
	ErrFailedToCopyDir      = errors.New("failed to copy directory")
	ErrFailedToCheckFile    = errors.New("failed to check file")
)
```


```go
package backup

import "errors"

var (
	ErrBackupOperationFailed = errors.New("backup operation failed")
	ErrMultipleBackupsFailed = errors.New("multiple backup operations failed")
)
```


Now update your implementation files to use these errors:

```go
// ... existing code ...

func (d *DefaultFilesHandler) CreateDirIfNotExists(path string) error {
	cleanedPath := filepath.Clean(path)
	if !filepath.IsAbs(cleanedPath) {
		return fmt.Errorf("%w: %q. Please use an absolute path", ErrPathNotAbsolute, path)
	}
	if err := d.stdlib.MkdirAll(cleanedPath, defaultDirPerms); err != nil {
		return fmt.Errorf("%w %q: %w", ErrFailedToCreateDir, cleanedPath, err)
	}
	return nil
}

// ... existing code ...

func (d *DefaultFilesHandler) RequireFilesInWd(filenames ...string) error {
	if len(filenames) == 0 {
		return nil
	}

	wd, err := d.stdlib.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	var missing []string
	for _, filename := range filenames {
		fullPath := filepath.Join(wd, filepath.Clean(filename))
		if _, err := d.stdlib.Stat(fullPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				missing = append(missing, filename)
			} else {
				return fmt.Errorf("%w %s: %w", ErrFailedToCheckFile, fullPath, err)
			}
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("%w: %s", ErrRequiredFileNotFound, strings.Join(missing, ", "))
	}
	return nil
}

func (d *DefaultFilesHandler) RequireDir(path string) error {
	cleanedPath := filepath.Clean(path)
	if !filepath.IsAbs(cleanedPath) {
		return fmt.Errorf("%w: %q. Please use an absolute path", ErrPathNotAbsolute, path)
	}
	
	info, err := d.stdlib.Stat(cleanedPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %s", ErrRequiredDirNotFound, cleanedPath)
		}
		return fmt.Errorf("%w %s: %w", ErrFailedToCheckFile, cleanedPath, err)
	}
	
	if !info.IsDir() {
		return fmt.Errorf("%w: %s", ErrNotADirectory, cleanedPath)
	}
	return nil
}

// ... existing code ...
```


```go
// ... existing code ...

func (l *LocalBackupList) RunAll() error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(l.backups))
	for _, operation := range l.backups {
		wg.Add(1)
		go func(op LocalBackup) {
			defer wg.Done()
			if _, err := op.Run(); err != nil {
				errChan <- fmt.Errorf("%w: %w", ErrBackupOperationFailed, err)
			}
		}(operation)
	}

	wg.Wait()
	close(errChan)

	// Collect all errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) != 0 {
		return fmt.Errorf("%w (%d): %v", ErrMultipleBackupsFailed, len(errs), errs)
	}

	return nil
}
```


Now update your tests:

```go
// ... existing code ...

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
	// Optionally still check the message contains the filename
	if !strings.Contains(err.Error(), "missing.txt") {
		t.Errorf("expected error message to contain filename, got: %v", err)
	}
}

// ... existing code ...
```


```go
// ... existing code ...

func TestLocalBackupList_RunAll_SecondAndThirdFail(t *testing.T) {
	var executionCount atomic.Int32
	errorFrom2 := errors.New("backup 2 crashed")
	errorFrom3 := errors.New("backup 3 crashed")
	list := NewLocalBackupList()
	// First backup succeeds
	list.Add(&mockLocalBackup{
		runFunc: func() (string, error) {
			executionCount.Add(1)
			return "/backup/1", nil
		},
	})
	// Second backup fails
	list.Add(&mockLocalBackup{
		runFunc: func() (string, error) {
			executionCount.Add(1)
			return "", errorFrom2
		},
	})
	// Third backup fails
	list.Add(&mockLocalBackup{
		runFunc: func() (string, error) {
			executionCount.Add(1)
			return "", errorFrom3
		},
	})

	err := list.RunAll()

	if executionCount.Load() != 3 {
		t.Errorf("expected 3 backups to execute, got %d", executionCount.Load())
	}
	if err == nil {
		t.Fatal("expected error when backups fail, got nil")
	}
	
	// Check for the sentinel error
	if !errors.Is(err, ErrMultipleBackupsFailed) {
		t.Errorf("expected ErrMultipleBackupsFailed, got: %v", err)
	}
	
	// Optionally verify wrapped errors
	if !errors.Is(err, errorFrom2) {
		t.Errorf("expected error to contain backup 2 error")
	}
	if !errors.Is(err, errorFrom3) {
		t.Errorf("expected error to contain backup 3 error")
	}
}

// ... existing code ...
```


## Benefits of This Approach

✅ **Standard Go Practice**: This is the idiomatic way to handle errors in Go  
✅ **Type-Safe**: Use `errors.Is()` instead of string comparison  
✅ **Maintainable**: Change error messages without breaking tests  
✅ **Wrappable**: Can wrap errors with `%w` to preserve the error chain  
✅ **Documentable**: Error variables can have doc comments explaining when they occur

## Is This Standard?

**Yes!** This is the recommended approach since Go 1.13 (2019). The Go standard library extensively uses this pattern:
- `io.EOF`
- `os.ErrNotExist`
- `context.Canceled`
- etc.

This approach gives you the flexibility to change error messages for better UX while keeping your tests stable and focused on behavior rather than exact wording.

-----

# Custom Error Types in Go

The second approach is to create **custom error types** - structs that implement the `error` interface. This is useful when you need to attach **additional context or data** to your errors beyond just a message.

## When to Use Custom Error Types

Use custom error types when you need to:
- Store additional data with the error (paths, counts, lists, etc.)
- Provide structured error information
- Perform type assertions to extract error details
- Have complex error handling logic

## Basic Implementation

Here's how you'd implement custom error types for your scenario:

```go
package system

import (
	"fmt"
	"strings"
)

// PathError represents an error related to file paths
type PathError struct {
	Op   string // Operation: "create", "check", "remove", etc.
	Path string // The path that caused the error
	Err  error  // The underlying error
}

func (e *PathError) Error() string {
	return fmt.Sprintf("%s %q: %v", e.Op, e.Path, e.Err)
}

func (e *PathError) Unwrap() error {
	return e.Err
}

// MissingFilesError represents one or more missing required files
type MissingFilesError struct {
	Files []string // List of missing filenames
}

func (e *MissingFilesError) Error() string {
	return fmt.Sprintf("required files not found: %s", strings.Join(e.Files, ", "))
}

// Count returns the number of missing files
func (e *MissingFilesError) Count() int {
	return len(e.Files)
}

// NotAbsolutePathError indicates a relative path was used where absolute is required
type NotAbsolutePathError struct {
	Path string
}

func (e *NotAbsolutePathError) Error() string {
	return fmt.Sprintf("path is not absolute: %q. Please use an absolute path", e.Path)
}

// NotDirectoryError indicates a path exists but is not a directory
type NotDirectoryError struct {
	Path string
}

func (e *NotDirectoryError) Error() string {
	return fmt.Sprintf("path %q is not a directory", e.Path)
}
```


```go
package backup

import "fmt"

// BackupError represents an error from a single backup operation
type BackupError struct {
	BackupPath string // Optional: path where backup was attempted
	Err        error  // The underlying error
}

func (e *BackupError) Error() string {
	if e.BackupPath != "" {
		return fmt.Sprintf("backup operation failed for %q: %v", e.BackupPath, e.Err)
	}
	return fmt.Sprintf("backup operation failed: %v", e.Err)
}

func (e *BackupError) Unwrap() error {
	return e.Err
}

// MultipleBackupErrors represents multiple failed backup operations
type MultipleBackupErrors struct {
	Errors []error
}

func (e *MultipleBackupErrors) Error() string {
	return fmt.Sprintf("%d backup operations failed: %v", len(e.Errors), e.Errors)
}

// Count returns the number of failed backups
func (e *MultipleBackupErrors) Count() int {
	return len(e.Errors)
}

// Unwrap returns the slice of errors (for Go 1.20+)
func (e *MultipleBackupErrors) Unwrap() []error {
	return e.Errors
}
```


## Using Custom Error Types in Your Code

```go
// ... existing code ...

func (d *DefaultFilesHandler) CreateDirIfNotExists(path string) error {
	cleanedPath := filepath.Clean(path)
	if !filepath.IsAbs(cleanedPath) {
		return &NotAbsolutePathError{Path: path}
	}
	if err := d.stdlib.MkdirAll(cleanedPath, defaultDirPerms); err != nil {
		return &PathError{
			Op:   "create directory",
			Path: cleanedPath,
			Err:  err,
		}
	}
	return nil
}

func (d *DefaultFilesHandler) RequireFilesInWd(filenames ...string) error {
	if len(filenames) == 0 {
		return nil
	}

	wd, err := d.stdlib.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	var missing []string
	for _, filename := range filenames {
		fullPath := filepath.Join(wd, filepath.Clean(filename))
		if _, err := d.stdlib.Stat(fullPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				missing = append(missing, filename)
			} else {
				return &PathError{
					Op:   "check file",
					Path: fullPath,
					Err:  err,
				}
			}
		}
	}

	if len(missing) > 0 {
		return &MissingFilesError{Files: missing}
	}
	return nil
}

func (d *DefaultFilesHandler) RequireDir(path string) error {
	cleanedPath := filepath.Clean(path)
	if !filepath.IsAbs(cleanedPath) {
		return &NotAbsolutePathError{Path: path}
	}
	
	info, err := d.stdlib.Stat(cleanedPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &PathError{
				Op:   "find directory",
				Path: cleanedPath,
				Err:  os.ErrNotExist,
			}
		}
		return &PathError{
			Op:   "check directory",
			Path: cleanedPath,
			Err:  err,
		}
	}
	
	if !info.IsDir() {
		return &NotDirectoryError{Path: cleanedPath}
	}
	return nil
}

// ... existing code ...
```


```go
// ... existing code ...

func (l *LocalBackupList) RunAll() error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(l.backups))
	for _, operation := range l.backups {
		wg.Add(1)
		go func(op LocalBackup) {
			defer wg.Done()
			if _, err := op.Run(); err != nil {
				errChan <- &BackupError{Err: err}
			}
		}(operation)
	}

	wg.Wait()
	close(errChan)

	// Collect all errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) != 0 {
		return &MultipleBackupErrors{Errors: errs}
	}

	return nil
}
```


## Testing with Custom Error Types

```go
// ... existing code ...

func TestDefaultFilesHandler_CreateDirIfNotExists_PathNotAbs(t *testing.T) {
	files := &DefaultFilesHandler{stdlib: &mockStdlib{}}

	err := files.CreateDirIfNotExists("relative/path")

	if err == nil {
		t.Fatal("expected error when path is relative, got nil")
	}
	
	// Type assertion to get the custom error
	var pathErr *NotAbsolutePathError
	if !errors.As(err, &pathErr) {
		t.Fatalf("expected NotAbsolutePathError, got: %T", err)
	}
	
	// Now you can access the error's fields
	if pathErr.Path != "relative/path" {
		t.Errorf("expected path %q, got %q", "relative/path", pathErr.Path)
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
	
	// Type assertion to extract structured data
	var pathErr *PathError
	if !errors.As(err, &pathErr) {
		t.Fatalf("expected PathError, got: %T", err)
	}
	
	if pathErr.Op != "create directory" {
		t.Errorf("expected operation %q, got %q", "create directory", pathErr.Op)
	}
	if pathErr.Path != "/home/user/restricteddir" {
		t.Errorf("expected path %q, got %q", "/home/user/restricteddir", pathErr.Path)
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v", expectedErr)
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
	
	// Type assertion to access structured data
	var missingErr *MissingFilesError
	if !errors.As(err, &missingErr) {
		t.Fatalf("expected MissingFilesError, got: %T", err)
	}
	
	// Access the Count method
	if missingErr.Count() != 2 {
		t.Errorf("expected 2 missing files, got %d", missingErr.Count())
	}
	
	// Check the actual file list
	expectedFiles := []string{"missing1.txt", "missing2.txt"}
	if diff := cmp.Diff(expectedFiles, missingErr.Files); diff != "" {
		t.Errorf("missing files mismatch (-want +got):\n%s", diff)
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
		t.Fatal("expected error when path is not a directory, got nil")
	}
	
	var notDirErr *NotDirectoryError
	if !errors.As(err, &notDirErr) {
		t.Fatalf("expected NotDirectoryError, got: %T", err)
	}
	
	if notDirErr.Path != "/home/user/file.txt" {
		t.Errorf("expected path %q, got %q", "/home/user/file.txt", notDirErr.Path)
	}
}

// ... existing code ...
```


```go
// ... existing code ...

func TestLocalBackupList_RunAll_SecondAndThirdFail(t *testing.T) {
	var executionCount atomic.Int32
	errorFrom2 := errors.New("backup 2 crashed")
	errorFrom3 := errors.New("backup 3 crashed")
	list := NewLocalBackupList()
	
	list.Add(&mockLocalBackup{
		runFunc: func() (string, error) {
			executionCount.Add(1)
			return "/backup/1", nil
		},
	})
	list.Add(&mockLocalBackup{
		runFunc: func() (string, error) {
			executionCount.Add(1)
			return "", errorFrom2
		},
	})
	list.Add(&mockLocalBackup{
		runFunc: func() (string, error) {
			executionCount.Add(1)
			return "", errorFrom3
		},
	})

	err := list.RunAll()

	if executionCount.Load() != 3 {
		t.Errorf("expected 3 backups to execute, got %d", executionCount.Load())
	}
	if err == nil {
		t.Fatal("expected error when backups fail, got nil")
	}
	
	// Type assertion to access the custom error type
	var multiErr *MultipleBackupErrors
	if !errors.As(err, &multiErr) {
		t.Fatalf("expected MultipleBackupErrors, got: %T", err)
	}
	
	// Use the Count method
	if multiErr.Count() != 2 {
		t.Errorf("expected 2 failed backups, got %d", multiErr.Count())
	}
	
	// Access the individual errors
	if len(multiErr.Errors) != 2 {
		t.Fatalf("expected 2 errors in the list, got %d", len(multiErr.Errors))
	}
	
	// Check that each error wraps the original errors
	foundError2 := false
	foundError3 := false
	for _, e := range multiErr.Errors {
		if errors.Is(e, errorFrom2) {
			foundError2 = true
		}
		if errors.Is(e, errorFrom3) {
			foundError3 = true
		}
	}
	
	if !foundError2 {
		t.Error("expected to find error from backup 2")
	}
	if !foundError3 {
		t.Error("expected to find error from backup 3")
	}
}

func TestLocalBackupList_RunAll_AllFail(t *testing.T) {
	list := NewLocalBackupList()
	for i := 1; i <= 3; i++ {
		backupNum := i
		list.Add(&mockLocalBackup{
			runFunc: func() (string, error) {
				return "", errors.New(fmt.Sprintf("backup %d crashed", backupNum))
			},
		})
	}

	err := list.RunAll()

	if err == nil {
		t.Fatal("expected error when all backups fail, got nil")
	}
	
	var multiErr *MultipleBackupErrors
	if !errors.As(err, &multiErr) {
		t.Fatalf("expected MultipleBackupErrors, got: %T", err)
	}
	
	if multiErr.Count() != 3 {
		t.Errorf("expected 3 failed backups, got %d", multiErr.Count())
	}
}

// ... existing code ...
```


## Comparison: Sentinel Errors vs Custom Error Types

| Feature | Sentinel Errors | Custom Error Types |
|---------|----------------|-------------------|
| **Simplicity** | ✅ Very simple | ⚠️ More complex |
| **Structured Data** | ❌ No extra data | ✅ Can store fields |
| **Testing** | ✅ `errors.Is()` | ✅ `errors.As()` + field access |
| **Use Case** | Simple error conditions | Complex error information |
| **Standard Library** | Used extensively | Used for specific cases (e.g., `os.PathError`) |

## When to Choose Each Approach

### Use Sentinel Errors When:
- You just need to identify the error type
- No additional context is needed beyond the message
- Maximum simplicity is desired
- **Most common use case** ✅

### Use Custom Error Types When:
- You need to store structured data (paths, counts, lists)
- Callers might need to extract error details programmatically
- You want to provide methods on errors (like `Count()`)
- You're building a library and want rich error information

## Real-World Examples from Go Standard Library

**Sentinel Errors:**
```go
io.EOF
os.ErrNotExist
context.Canceled
```


**Custom Error Types:**
```go
*os.PathError    // Has Op, Path, Err fields
*net.OpError     // Has Op, Net, Source, Addr, Err fields
*json.SyntaxError // Has msg and Offset fields
```


Both approaches are idiomatic Go and widely used. For your project, I'd recommend starting with **sentinel errors** for simplicity, and only moving to custom error types if you find yourself needing to access structured error data programmatically.

-----

# Previous code failing because of Go's error unwrapping

The code above regarding `TestLocalBackupList_RunAll_SecondAndThirdFail` was failing. Below is the AI's answer when asked for it.
I think this answer is wrong, `Errorf` seems to be wrapping ALL the `%w` you pass in, the problem seems to be that we were
passing an `[]error` into `Errorf`'s `%v`: using `%w` with `[]error` was giving the following error (I assume it would
have thrown it when building the project):
```text
 fmt.Errorf format %w has arg errs of wrong type []error
```

Great question! The issue is that **`errors.Is()` only unwraps one level at a time** using the `Unwrap()` method, and your current implementation doesn't properly preserve the error chain.

## Why It's Failing

Let's trace what happens:

1. **Individual backup fails**: Returns `errorFrom2`
2. **In goroutine**: Wraps it as `fmt.Errorf("%w: %w", ErrBackupOperationFailed, errorFrom2)`
    - This creates an error that wraps `ErrBackupOperationFailed`, but **loses** `errorFrom2` because `fmt.Errorf` with `%w` can only wrap ONE error (the first `%w`)
3. **In RunAll**: Wraps the slice as `fmt.Errorf("%w (%d): %v", ErrMultipleBackupOperationsFailed, len(errs), errs)`
    - The `%v` doesn't preserve the error chain at all

So when you call `errors.Is(err, errorFrom2)`, it can't find `errorFrom2` in the chain because it was lost during wrapping.

## The Problem with `fmt.Errorf` and Multiple `%w`

```go
// ❌ WRONG: Only the FIRST %w is wrapped, second is treated as %v
fmt.Errorf("%w: %w", ErrBackupOperationFailed, err)
// Result: Only wraps ErrBackupOperationFailed

// ❌ WRONG: %v doesn't preserve error wrapping
fmt.Errorf("%w (%d): %v", ErrMultipleBackupOperationsFailed, len(errs), errs)
// Result: Only wraps ErrMultipleBackupOperationsFailed
```


## Solution 1: Use Custom Error Type with `Unwrap() []error` (Go 1.20+)

This is the **modern, recommended approach** for handling multiple errors:

```go
package backup

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrBackupOperationFailed = errors.New("backup operation failed")
)

// MultipleBackupErrors represents multiple failed backup operations
type MultipleBackupErrors struct {
	Errors []error
}

func (e *MultipleBackupErrors) Error() string {
	var messages []string
	for _, err := range e.Errors {
		messages = append(messages, err.Error())
	}
	return fmt.Sprintf("%d backup operations failed: [%s]",
		len(e.Errors),
		strings.Join(messages, "; "))
}

// Unwrap returns the slice of errors (Go 1.20+ feature)
// This allows errors.Is() to check against ALL wrapped errors
func (e *MultipleBackupErrors) Unwrap() []error {
	return e.Errors
}

// Count returns the number of failed backups
func (e *MultipleBackupErrors) Count() int {
	return len(e.Errors)
}
```


```go
package backup

import (
	"fmt"
	"sync"
)

type LocalBackupList struct {
	backups []LocalBackup
}

func NewLocalBackupList() *LocalBackupList {
	return &LocalBackupList{
		backups: []LocalBackup{},
	}
}

// Add adds a backup operation to the list
func (l *LocalBackupList) Add(backup LocalBackup) {
	l.backups = append(l.backups, backup)
}

// RunAll runs all backup operations concurrently
func (l *LocalBackupList) RunAll() error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(l.backups))
	for _, operation := range l.backups {
		wg.Add(1)
		go func(op LocalBackup) {
			defer wg.Done()
			if _, err := op.Run(); err != nil {
				// Wrap with context but preserve the original error
				errChan <- fmt.Errorf("%w: %w", ErrBackupOperationFailed, err)
			}
		}(operation)
	}

	wg.Wait()
	close(errChan)

	// Collect all errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) != 0 {
		// Return custom error type that properly unwraps all errors
		return &MultipleBackupErrors{Errors: errs}
	}

	return nil
}
```


Now `errors.Is()` will work because `MultipleBackupErrors.Unwrap()` returns a slice, and the Go standard library will check each error in the slice.

## Solution 2: Use `errors.Join()` (Go 1.20+ - Even Simpler)

Go 1.20 introduced `errors.Join()` which is purpose-built for this:

```go
package backup

import (
	"errors"
	"fmt"
	"sync"
)

var (
	ErrBackupOperationFailed          = errors.New("backup operation failed")
	ErrMultipleBackupOperationsFailed = errors.New("multiple backup operations failed")
)

type LocalBackupList struct {
	backups []LocalBackup
}

func NewLocalBackupList() *LocalBackupList {
	return &LocalBackupList{
		backups: []LocalBackup{},
	}
}

// Add adds a backup operation to the list
func (l *LocalBackupList) Add(backup LocalBackup) {
	l.backups = append(l.backups, backup)
}

// RunAll runs all backup operations concurrently
func (l *LocalBackupList) RunAll() error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(l.backups))
	for _, operation := range l.backups {
		wg.Add(1)
		go func(op LocalBackup) {
			defer wg.Done()
			if _, err := op.Run(); err != nil {
				// Wrap with context but preserve the original error
				errChan <- fmt.Errorf("%w: %w", ErrBackupOperationFailed, err)
			}
		}(operation)
	}

	wg.Wait()
	close(errChan)

	// Collect all errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) != 0 {
		// Join all errors together - this creates an error that properly unwraps
		joinedErr := errors.Join(errs...)
		// Wrap with a sentinel error for identification
		return fmt.Errorf("%w (%d operations): %w", 
			ErrMultipleBackupOperationsFailed, 
			len(errs), 
			joinedErr)
	}

	return nil
}
```


With `errors.Join()`, the returned error has an `Unwrap() []error` method automatically, so `errors.Is()` will traverse all the joined errors.

## Updated Test

Your test should now pass without modification:

```go
func TestLocalBackupList_RunAll_SecondAndThirdFail(t *testing.T) {
	var executionCount atomic.Int32
	errorFrom2 := errors.New("backup 2 crashed")
	errorFrom3 := errors.New("backup 3 crashed")
	list := NewLocalBackupList()
	
	// First backup succeeds
	list.Add(&mockLocalBackup{
		runFunc: func() (string, error) {
			executionCount.Add(1)
			return "/backup/1", nil
		},
	})
	// Second backup fails
	list.Add(&mockLocalBackup{
		runFunc: func() (string, error) {
			executionCount.Add(1)
			return "", errorFrom2
		},
	})
	// Third backup fails
	list.Add(&mockLocalBackup{
		runFunc: func() (string, error) {
			executionCount.Add(1)
			return "", errorFrom3
		},
	})

	err := list.RunAll()

	if executionCount.Load() != 3 {
		t.Errorf("expected 3 backups to execute, got %d", executionCount.Load())
	}
	if err == nil {
		t.Fatal("expected error when backups fail, got nil")
	}
	
	// Check for sentinel error
	if !errors.Is(err, ErrMultipleBackupOperationsFailed) {
		t.Errorf("expected ErrMultipleBackupOperationsFailed, got: %v", err)
	}
	
	// ✅ These will now work!
	if !errors.Is(err, errorFrom2) {
		t.Errorf("expected error to contain backup 2 error")
	}
	if !errors.Is(err, errorFrom3) {
		t.Errorf("expected error to contain backup 3 error")
	}
	
	// Also check the wrapping error
	if !errors.Is(err, ErrBackupOperationFailed) {
		t.Errorf("expected error to contain ErrBackupOperationFailed")
	}
}
```


## Key Takeaways

1. **`fmt.Errorf` with `%w` only wraps ONE error** - the first `%w` encountered
2. **Use `Unwrap() []error`** method to unwrap multiple errors (Go 1.20+)
3. **Use `errors.Join()`** for the simplest solution to combine multiple errors (Go 1.20+)
4. **Custom error types with `Unwrap() []error`** give you more control over error structure

## Which Solution to Choose?

- **`errors.Join()`**: Use when you just need to combine errors simply ✅
- **Custom Error Type**: Use when you need structured data (counts, methods, etc.) ✅

Both work perfectly with `errors.Is()` and `errors.As()` for proper error chain traversal!