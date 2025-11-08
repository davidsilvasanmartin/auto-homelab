package backup

import (
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

// mockLocalBackup is a mock implementation of LocalBackup for testing
type mockLocalBackup struct {
	runFunc func() error
}

func (m *mockLocalBackup) Run() error {
	if m.runFunc != nil {
		return m.runFunc()
	}
	return nil
}

func TestLocalBackupList_RunAll_ThreeSuccessful(t *testing.T) {
	var executionCount atomic.Int32
	list := NewLocalBackupList()
	for i := 0; i < 3; i++ {
		list.Add(&mockLocalBackup{
			runFunc: func() error {
				// Simulate some work
				time.Sleep(10 * time.Nanosecond)
				executionCount.Add(1)
				return nil
			},
		})
	}

	err := list.RunAll()

	if err != nil {
		t.Errorf("expected no error when all backups succeed, got: %v", err)
	}
	if executionCount.Load() != 3 {
		t.Errorf("expected 3 backups to execute, got %d", executionCount.Load())
	}
}

func TestLocalBackupList_RunAll_SecondAndThirdFail(t *testing.T) {
	var executionCount atomic.Int32
	errorFrom2 := errors.New("backup 2 crashed")
	errorFrom3 := errors.New("backup 3 crashed")
	list := NewLocalBackupList()
	// First backup succeeds
	list.Add(&mockLocalBackup{
		runFunc: func() error {
			executionCount.Add(1)
			return nil
		},
	})
	// Second backup fails
	list.Add(&mockLocalBackup{
		runFunc: func() error {
			executionCount.Add(1)
			return errorFrom2
		},
	})
	// Third backup fails
	list.Add(&mockLocalBackup{
		runFunc: func() error {
			executionCount.Add(1)
			return errorFrom3
		},
	})

	err := list.RunAll()

	if executionCount.Load() != 3 {
		t.Errorf("expected 3 backups to execute, got %d", executionCount.Load())
	}
	if err == nil {
		t.Fatal("expected error when backups fail, got nil")
	}
	if !errors.Is(err, ErrMultipleBackupOperationsFailed) {
		t.Errorf("wrong error, got: %v", err)
	}
	if !errors.Is(err, errorFrom2) {
		t.Errorf("expected error to contain backup 2 error")
	}
	if !errors.Is(err, errorFrom3) {
		t.Errorf("expected error to contain backup 3 error")
	}
}

func TestLocalBackupList_RunAll_AllFail(t *testing.T) {
	list := NewLocalBackupList()
	for i := 1; i <= 3; i++ {
		backupNum := i
		list.Add(&mockLocalBackup{
			runFunc: func() error {
				return errors.New(fmt.Sprintf("backup %d crashed", backupNum))
			},
		})
	}

	err := list.RunAll()

	if err == nil {
		t.Fatal("expected error when all backups fail, got nil")
	}
	if !errors.Is(err, ErrMultipleBackupOperationsFailed) {
		t.Errorf("wrong error, got: %v", err)
	}
	if !errors.Is(err, ErrBackupOperationFailed) {
		t.Errorf("expected error to wrap individual errors, got: %v", err)
	}
}

func TestLocalBackupList_RunAll_EmptyList(t *testing.T) {
	list := NewLocalBackupList()

	err := list.RunAll()

	if err != nil {
		t.Errorf("expected no error for empty list, got: %v", err)
	}
}

func TestLocalBackupList_RunAll_ConcurrentExecution(t *testing.T) {
	list := NewLocalBackupList()
	// Add 3 backups, each taking 50ms
	for i := 0; i < 3; i++ {
		list.Add(&mockLocalBackup{
			runFunc: func() error {
				time.Sleep(50 * time.Millisecond)
				return nil
			},
		})
	}

	startTime := time.Now()
	err := list.RunAll()
	elapsed := time.Since(startTime)

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	// If they run sequentially, it would take at least 150ms. If concurrent, it should take a lot less
	// Note this test may fail on very slow systems.
	if elapsed > 120*time.Millisecond {
		t.Errorf("backups appear to run sequentially (took %v), expected concurrent execution", elapsed)
	}
}
