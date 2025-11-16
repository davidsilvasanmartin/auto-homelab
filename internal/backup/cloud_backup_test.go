package backup

import (
	"errors"
	"testing"
)

type mockResticClient struct {
	initFunc      func() error
	backupFunc    func(path string, tags []string) error
	forgetFunc    func(keepWithin string, prune bool) error
	checkFunc     func() error
	snapshotsFunc func() error
	listFilesFunc func(snapshotID string) error
	restoreFunc   func(targetDir string) error
}

func (m *mockResticClient) Init() error {
	if m.initFunc != nil {
		return m.initFunc()
	}
	return nil
}
func (m *mockResticClient) Backup(path string, tags []string) error {
	if m.backupFunc != nil {
		return m.backupFunc(path, tags)
	}
	return nil
}
func (m *mockResticClient) Forget(keepWithin string, prune bool) error {
	if m.forgetFunc != nil {
		return m.forgetFunc(keepWithin, prune)
	}
	return nil
}
func (m *mockResticClient) Check() error {
	if m.checkFunc != nil {
		return m.checkFunc()
	}
	return nil
}
func (m *mockResticClient) Snapshots() error {
	if m.snapshotsFunc != nil {
		return m.snapshotsFunc()
	}
	return nil
}
func (m *mockResticClient) ListFiles(snapshotID string) error {
	if m.listFilesFunc != nil {
		return m.listFilesFunc(snapshotID)
	}
	return nil
}
func (m *mockResticClient) Restore(targetDir string) error {
	if m.restoreFunc != nil {
		return m.restoreFunc(targetDir)
	}
	return nil
}

func TestCloudBackup_RunFullBackup_Success(t *testing.T) {
	initCalled := false
	backupCalled := false
	forgetCalled := false
	var capturedBackupPath string
	var capturedKeepWithin string
	var capturedPrune bool
	cloudBackup := &CloudBackup{
		client: &mockResticClient{
			initFunc: func() error {
				initCalled = true
				return nil
			},
			backupFunc: func(path string, tags []string) error {
				backupCalled = true
				capturedBackupPath = path
				return nil
			},
			forgetFunc: func(keepWithin string, prune bool) error {
				forgetCalled = true
				capturedKeepWithin = keepWithin
				capturedPrune = prune
				return nil
			},
		},
		files: &mockFilesHandler{},
		config: ResticConfig{
			BackupPath:    "/data/backup",
			RetentionDays: 30,
		},
	}

	err := cloudBackup.RunFullBackup()

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !initCalled {
		t.Error("expected Init to be called")
	}
	if !backupCalled {
		t.Error("expected Backup to be called")
	}
	if !forgetCalled {
		t.Error("expected Forget to be called")
	}
	if capturedBackupPath != "/data/backup" {
		t.Errorf("expected backup path %q, got %q", "/data/backup", capturedBackupPath)
	}
	if capturedKeepWithin != "30d" {
		t.Errorf("expected keepWithin %q, got %q", "30d", capturedKeepWithin)
	}
	if !capturedPrune {
		t.Error("expected prune to be true")
	}
}

func TestCloudBackup_RunFullBackup_TagsContainTimestamp(t *testing.T) {
	var capturedTags []string
	cloudBackup := &CloudBackup{
		client: &mockResticClient{
			initFunc: func() error {
				return nil
			},
			backupFunc: func(path string, tags []string) error {
				capturedTags = tags
				return nil
			},
			forgetFunc: func(keepWithin string, prune bool) error {
				return nil
			},
		},
		files: &mockFilesHandler{},
		config: ResticConfig{
			BackupPath:    "/data/backup",
			RetentionDays: 30,
		},
	}

	err := cloudBackup.RunFullBackup()

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(capturedTags) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(capturedTags))
	}
	// Tag should start with "automatic-"
	tag := capturedTags[0]
	if len(tag) < 10 || tag[:10] != "automatic-" {
		t.Errorf("expected tag to start with 'automatic-', got: %q", tag)
	}
}

func TestCloudBackup_RunFullBackup_InitFails(t *testing.T) {
	expectedErr := errors.New("init failed")
	backupCalled := false
	cloudBackup := &CloudBackup{
		client: &mockResticClient{
			initFunc: func() error {
				return expectedErr
			},
			backupFunc: func(path string, tags []string) error {
				backupCalled = true
				return nil
			},
		},
		files: &mockFilesHandler{},
		config: ResticConfig{
			BackupPath:    "/data/backup",
			RetentionDays: 30,
		},
	}

	err := cloudBackup.RunFullBackup()

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got: %v", expectedErr, err)
	}
	if backupCalled {
		t.Error("expected Backup NOT to be called when Init fails")
	}
}

func TestCloudBackup_RunFullBackup_BackupPathDoesNotExist(t *testing.T) {
	expectedErr := errors.New("path does not exist")
	backupCalled := false
	cloudBackup := &CloudBackup{
		client: &mockResticClient{
			initFunc: func() error {
				return nil
			},
			backupFunc: func(path string, tags []string) error {
				backupCalled = true
				return nil
			},
		},
		files: &mockFilesHandler{
			ensureDirExists: func(path string) error {
				return expectedErr
			},
		},
		config: ResticConfig{
			BackupPath:    "/data/backup",
			RetentionDays: 30,
		},
	}

	err := cloudBackup.RunFullBackup()

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got: %v", expectedErr, err)
	}
	if backupCalled {
		t.Error("expected Backup NOT to be called when path doesn't exist")
	}
}

func TestCloudBackup_RunFullBackup_BackupFails(t *testing.T) {
	expectedErr := errors.New("backup failed")
	forgetCalled := false
	cloudBackup := &CloudBackup{
		client: &mockResticClient{
			initFunc: func() error {
				return nil
			},
			backupFunc: func(path string, tags []string) error {
				return expectedErr
			},
			forgetFunc: func(keepWithin string, prune bool) error {
				forgetCalled = true
				return nil
			},
		},
		files: &mockFilesHandler{},
		config: ResticConfig{
			BackupPath:    "/data/backup",
			RetentionDays: 30,
		},
	}

	err := cloudBackup.RunFullBackup()

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got: %v", expectedErr, err)
	}
	if forgetCalled {
		t.Error("expected Forget NOT to be called when Backup fails")
	}
}

func TestCloudBackup_RunFullBackup_ForgetFails(t *testing.T) {
	expectedErr := errors.New("forget failed")
	cloudBackup := &CloudBackup{
		client: &mockResticClient{
			initFunc: func() error {
				return nil
			},
			backupFunc: func(path string, tags []string) error {
				return nil
			},
			forgetFunc: func(keepWithin string, prune bool) error {
				return expectedErr
			},
		},
		files: &mockFilesHandler{},
		config: ResticConfig{
			BackupPath:    "/data/backup",
			RetentionDays: 30,
		},
	}

	err := cloudBackup.RunFullBackup()

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got: %v", expectedErr, err)
	}
}

func TestCloudBackup_Init_Success(t *testing.T) {
	initCalled := false
	cloudBackup := &CloudBackup{
		client: &mockResticClient{
			initFunc: func() error {
				initCalled = true
				return nil
			},
		},
		files:  &mockFilesHandler{},
		config: ResticConfig{},
	}

	err := cloudBackup.Init()

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !initCalled {
		t.Error("expected Init to be called on client")
	}
}

func TestCloudBackup_Init_Error(t *testing.T) {
	expectedErr := errors.New("init failed")
	cloudBackup := &CloudBackup{
		client: &mockResticClient{
			initFunc: func() error {
				return expectedErr
			},
		},
		files:  &mockFilesHandler{},
		config: ResticConfig{},
	}

	err := cloudBackup.Init()

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got: %v", expectedErr, err)
	}
}

func TestCloudBackup_Check_Success(t *testing.T) {
	checkCalled := false
	cloudBackup := &CloudBackup{
		client: &mockResticClient{
			checkFunc: func() error {
				checkCalled = true
				return nil
			},
		},
		files:  &mockFilesHandler{},
		config: ResticConfig{},
	}

	err := cloudBackup.Check()

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !checkCalled {
		t.Error("expected Check to be called on client")
	}
}

func TestCloudBackup_Check_Error(t *testing.T) {
	expectedErr := errors.New("check failed")
	cloudBackup := &CloudBackup{
		client: &mockResticClient{
			checkFunc: func() error {
				return expectedErr
			},
		},
		files:  &mockFilesHandler{},
		config: ResticConfig{},
	}

	err := cloudBackup.Check()

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got: %v", expectedErr, err)
	}
}

func TestCloudBackup_ListSnapshots_Success(t *testing.T) {
	snapshotsCalled := false
	cloudBackup := &CloudBackup{
		client: &mockResticClient{
			snapshotsFunc: func() error {
				snapshotsCalled = true
				return nil
			},
		},
		files:  &mockFilesHandler{},
		config: ResticConfig{},
	}

	err := cloudBackup.ListSnapshots()

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !snapshotsCalled {
		t.Error("expected Snapshots to be called on client")
	}
}

func TestCloudBackup_ListSnapshots_Error(t *testing.T) {
	expectedErr := errors.New("snapshots failed")
	cloudBackup := &CloudBackup{
		client: &mockResticClient{
			snapshotsFunc: func() error {
				return expectedErr
			},
		},
		files:  &mockFilesHandler{},
		config: ResticConfig{},
	}

	err := cloudBackup.ListSnapshots()

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got: %v", expectedErr, err)
	}
}

func TestCloudBackup_Prune_Success(t *testing.T) {
	forgetCalled := false
	var capturedKeepWithin string
	var capturedPrune bool
	cloudBackup := &CloudBackup{
		client: &mockResticClient{
			forgetFunc: func(keepWithin string, prune bool) error {
				forgetCalled = true
				capturedKeepWithin = keepWithin
				capturedPrune = prune
				return nil
			},
		},
		files: &mockFilesHandler{},
		config: ResticConfig{
			RetentionDays: 14,
		},
	}

	err := cloudBackup.Prune()

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !forgetCalled {
		t.Error("expected Forget to be called on client")
	}
	if capturedKeepWithin != "14d" {
		t.Errorf("expected keepWithin %q, got %q", "14d", capturedKeepWithin)
	}
	if !capturedPrune {
		t.Error("expected prune to be true")
	}
}

func TestCloudBackup_Prune_Error(t *testing.T) {
	expectedErr := errors.New("prune failed")
	cloudBackup := &CloudBackup{
		client: &mockResticClient{
			forgetFunc: func(keepWithin string, prune bool) error {
				return expectedErr
			},
		},
		files: &mockFilesHandler{},
		config: ResticConfig{
			RetentionDays: 14,
		},
	}

	err := cloudBackup.Prune()

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got: %v", expectedErr, err)
	}
}

func TestCloudBackup_Restore_Success(t *testing.T) {
	restoreCalled := false
	createDirCalled := false
	var capturedTargetDir string
	var capturedCreateDir string
	cloudBackup := &CloudBackup{
		client: &mockResticClient{
			restoreFunc: func(targetDir string) error {
				restoreCalled = true
				capturedTargetDir = targetDir
				return nil
			},
		},
		files: &mockFilesHandler{
			createDirIfNotExists: func(path string) error {
				createDirCalled = true
				capturedCreateDir = path
				return nil
			},
		},
		config: ResticConfig{},
	}

	err := cloudBackup.Restore("/restore/target")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !restoreCalled {
		t.Error("expected Restore to be called on client")
	}
	if !createDirCalled {
		t.Error("expected CreateDirIfNotExists to be called")
	}
	if capturedTargetDir != "/restore/target" {
		t.Errorf("expected target dir %q, got %q", "/restore/target", capturedTargetDir)
	}
	if capturedCreateDir != "/restore/target" {
		t.Errorf("expected create dir %q, got %q", "/restore/target", capturedCreateDir)
	}
}

func TestCloudBackup_Restore_GetAbsPathError(t *testing.T) {
	expectedErr := errors.New("get abs path failed")
	createDirCalled := false
	restoreCalled := false
	cloudBackup := &CloudBackup{
		client: &mockResticClient{
			restoreFunc: func(targetDir string) error {
				restoreCalled = true
				return nil
			},
		},
		files: &mockFilesHandler{
			getAbsPath: func(path string) (string, error) {
				return "", expectedErr
			},
			createDirIfNotExists: func(path string) error {
				createDirCalled = true
				return nil
			},
		},
		config: ResticConfig{},
	}

	err := cloudBackup.Restore("/restore/target")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got: %v", expectedErr, err)
	}
	if createDirCalled {
		t.Error("expected CreateDir NOT to be called when GetAbsPath fails")
	}
	if restoreCalled {
		t.Error("expected Restore NOT to be called when GetAbsPath fails")
	}
}

func TestCloudBackup_Restore_CreateDirError(t *testing.T) {
	expectedErr := errors.New("create dir failed")
	restoreCalled := false
	cloudBackup := &CloudBackup{
		client: &mockResticClient{
			restoreFunc: func(targetDir string) error {
				restoreCalled = true
				return nil
			},
		},
		files: &mockFilesHandler{
			createDirIfNotExists: func(path string) error {
				return expectedErr
			},
		},
		config: ResticConfig{},
	}

	err := cloudBackup.Restore("/restore/target")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got: %v", expectedErr, err)
	}
	if restoreCalled {
		t.Error("expected Restore NOT to be called when CreateDirIfNotExists fails")
	}
}

func TestCloudBackup_Restore_RestoreError(t *testing.T) {
	expectedErr := errors.New("restore failed")
	cloudBackup := &CloudBackup{
		client: &mockResticClient{
			restoreFunc: func(targetDir string) error {
				return expectedErr
			},
		},
		files:  &mockFilesHandler{},
		config: ResticConfig{},
	}

	err := cloudBackup.Restore("/restore/target")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got: %v", expectedErr, err)
	}
}

func TestCloudBackup_ListFiles_Success(t *testing.T) {
	listFilesCalled := false
	var capturedSnapshotID string
	cloudBackup := &CloudBackup{
		client: &mockResticClient{
			listFilesFunc: func(snapshotID string) error {
				listFilesCalled = true
				capturedSnapshotID = snapshotID
				return nil
			},
		},
		files:  &mockFilesHandler{},
		config: ResticConfig{},
	}

	err := cloudBackup.ListFiles("snapshot123")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !listFilesCalled {
		t.Error("expected ListFiles to be called on client")
	}
	if capturedSnapshotID != "snapshot123" {
		t.Errorf("expected snapshot ID %q, got %q", "snapshot123", capturedSnapshotID)
	}
}

func TestCloudBackup_ListFiles_Error(t *testing.T) {
	expectedErr := errors.New("list files failed")
	cloudBackup := &CloudBackup{
		client: &mockResticClient{
			listFilesFunc: func(snapshotID string) error {
				return expectedErr
			},
		},
		files:  &mockFilesHandler{},
		config: ResticConfig{},
	}

	err := cloudBackup.ListFiles("snapshot123")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got: %v", expectedErr, err)
	}
}
