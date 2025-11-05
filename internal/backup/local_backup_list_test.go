package backup

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// mockLocalBackup is a mock implementation of LocalBackup for testing
type mockLocalBackup struct {
	runFunc func() (string, error)
}

func (m *mockLocalBackup) Run() (string, error) {
	if m.runFunc != nil {
		return m.runFunc()
	}
	return "", nil
}

func TestLocalBackupList_RunAll_ThreeSuccessful(t *testing.T) {
	var executionCount atomic.Int32
	list := NewLocalBackupList()
	for i := 0; i < 3; i++ {
		list.Add(&mockLocalBackup{
			runFunc: func() (string, error) {
				// Simulate some work
				time.Sleep(10 * time.Nanosecond)
				executionCount.Add(1)
				return "/backup/path", nil
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
	firstError := errors.New("backup 2 failed")
	secondError := errors.New("backup 3 failed")
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
			return "", firstError
		},
	})
	// Third backup fails
	list.Add(&mockLocalBackup{
		runFunc: func() (string, error) {
			executionCount.Add(1)
			return "", secondError
		},
	})

	err := list.RunAll()

	if executionCount.Load() != 3 {
		t.Errorf("expected 3 backups to execute, got %d", executionCount.Load())
	}
	if err == nil {
		t.Fatal("expected error when backups fail, got nil")
	}
	expectedMsgParts := []string{
		"2 backup operations failed:",
		"backup operation failed: backup 2",
		"backup operation failed: backup 3",
	}
	actualMsg := err.Error()
	for _, expectedMsg := range expectedMsgParts {
		if !strings.Contains(actualMsg, expectedMsg) {
			t.Errorf("expected error message to contain %q, got %q", expectedMsg, actualMsg)
		}
	}
}

// TODO BELOW
//// TestLocalBackupList_RunAll_AllFail tests when all backups fail
//func TestLocalBackupList_RunAll_AllFail(t *testing.T) {
//	list := NewLocalBackupList()
//
//	for i := 1; i <= 3; i++ {
//		backupNum := i
//		list.Add(&mockLocalBackup{
//			runFunc: func() (string, error) {
//				return "", errors.New("backup failed")
//			},
//		})
//	}
//
//	err := list.RunAll()
//
//	if err == nil {
//		t.Fatal("expected error when all backups fail, got nil")
//	}
//
//	expectedPrefix := "backup operation failed:"
//	if len(err.Error()) < len(expectedPrefix) || err.Error()[:len(expectedPrefix)] != expectedPrefix {
//		t.Errorf("expected error message to start with %q, got: %q", expectedPrefix, err.Error())
//	}
//}
//
//// TestLocalBackupList_RunAll_EmptyList tests empty backup list
//func TestLocalBackupList_RunAll_EmptyList(t *testing.T) {
//	list := NewLocalBackupList()
//
//	err := list.RunAll()
//
//	if err != nil {
//		t.Errorf("expected no error for empty list, got: %v", err)
//	}
//}
//
//// TestLocalBackupList_Add_MultipleBackups tests adding multiple backups
//func TestLocalBackupList_Add_MultipleBackups(t *testing.T) {
//	list := NewLocalBackupList()
//
//	backup1 := &mockLocalBackup{}
//	backup2 := &mockLocalBackup{}
//	backup3 := &mockLocalBackup{}
//
//	list.Add(backup1)
//	list.Add(backup2)
//	list.Add(backup3)
//
//	if len(list.backups) != 3 {
//		t.Errorf("expected 3 backups in list, got %d", len(list.backups))
//	}
//}
//
//// TestLocalBackupList_RunAll_ConcurrentExecution verifies backups run concurrently
//func TestLocalBackupList_RunAll_ConcurrentExecution(t *testing.T) {
//	list := NewLocalBackupList()
//	startTime := time.Now()
//
//	// Add 3 backups that each take 50ms
//	for i := 0; i < 3; i++ {
//		list.Add(&mockLocalBackup{
//			runFunc: func() (string, error) {
//				time.Sleep(50 * time.Millisecond)
//				return "", nil
//			},
//		})
//	}
//
//	err := list.RunAll()
//
//	elapsed := time.Since(startTime)
//
//	if err != nil {
//		t.Errorf("expected no error, got: %v", err)
//	}
//
//	// If they ran sequentially, it would take 150ms+
//	// If concurrent, should complete in ~50-100ms
//	if elapsed > 120*time.Millisecond {
//		t.Errorf("backups appear to run sequentially (took %v), expected concurrent execution", elapsed)
//	}
//}
