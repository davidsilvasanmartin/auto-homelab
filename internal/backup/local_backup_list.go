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
	// TODO review and understand this code (thanks Claude :))
	var wg sync.WaitGroup
	errChan := make(chan error, len(l.backups))
	for _, operation := range l.backups {
		wg.Add(1)
		go func(op LocalBackup) {
			defer wg.Done()
			if _, err := op.Run(); err != nil {
				errChan <- fmt.Errorf("backup operation failed: %w", err)
			}
		}(operation)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}
