package backup

import (
	"errors"
	"testing"

	"github.com/davidsilvasanmartin/auto-homelab/internal/system"
)

func TestDefaultResticClient_Init_RepositoryExists(t *testing.T) {
	var executedCmd string
	client := &DefaultResticClient{
		commands: &mockCommands{
			execShellCommand: func(cmd string) system.RunnableCommand {
				executedCmd = cmd
				return &mockRunnableCommand{
					runFunc: func() error {
						// Snapshots command succeeds, repository exists
						return nil
					},
				}
			},
		},
		textFormatter: &mockTextFormatter{},
		config: ResticConfig{
			RepositoryURL:    "b2:b:p",
			B2KeyID:          "k1",
			B2ApplicationKey: "a2",
			ResticPassword:   "p3",
		},
	}

	err := client.Init()

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	expectedCmd := "RESTIC_REPOSITORY='b2:b:p' B2_ACCOUNT_ID='k1' B2_ACCOUNT_KEY='a2' RESTIC_PASSWORD='p3' restic snapshots"
	if executedCmd != expectedCmd {
		t.Errorf("expected command to be %q, got: %q", expectedCmd, executedCmd)
	}
}

func TestDefaultResticClient_Init_RepositoryDoesNotExist(t *testing.T) {
	callCount := 0
	var lastCmd string
	client := &DefaultResticClient{
		commands: &mockCommands{
			execShellCommand: func(cmd string) system.RunnableCommand {
				callCount++
				lastCmd = cmd
				return &mockRunnableCommand{
					runFunc: func() error {
						if callCount == 1 {
							// First call (snapshots) fails - repository doesn't exist
							return errors.New("repository not found")
						}
						// Second call (init) succeeds
						return nil
					},
				}
			},
		},
		textFormatter: &mockTextFormatter{},
		config: ResticConfig{
			RepositoryURL:    "b2:b:p",
			B2KeyID:          "k1",
			B2ApplicationKey: "a2",
			ResticPassword:   "p3",
		},
	}

	err := client.Init()

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 commands to be executed, got %d", callCount)
	}
	expectedCmd := "RESTIC_REPOSITORY='b2:b:p' B2_ACCOUNT_ID='k1' B2_ACCOUNT_KEY='a2' RESTIC_PASSWORD='p3' restic init"
	if lastCmd != expectedCmd {
		t.Errorf("expected last command to be %q, got: %q", expectedCmd, lastCmd)
	}
}

func TestDefaultResticClient_Init_InitFails(t *testing.T) {
	expectedErr := errors.New("init failed")
	callCount := 0
	client := &DefaultResticClient{
		commands: &mockCommands{
			execShellCommand: func(cmd string) system.RunnableCommand {
				callCount++
				return &mockRunnableCommand{
					runFunc: func() error {
						if callCount == 1 {
							// Snapshots fails
							return errors.New("repository not found")
						}
						// Init fails
						return expectedErr
					},
				}
			},
		},
		textFormatter: &mockTextFormatter{},
		config: ResticConfig{
			RepositoryURL:    "b2:b:p",
			B2KeyID:          "k1",
			B2ApplicationKey: "a2",
			ResticPassword:   "p3",
		},
	}

	err := client.Init()

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to be %v, got: %v", expectedErr, err)
	}
}

func TestDefaultResticClient_Backup_Success(t *testing.T) {
	var executedCmd string
	client := &DefaultResticClient{
		commands: &mockCommands{
			execShellCommand: func(cmd string) system.RunnableCommand {
				executedCmd = cmd
				return &mockRunnableCommand{}
			},
		},
		textFormatter: &mockTextFormatter{},
		config: ResticConfig{
			RepositoryURL:    "b2:b:p",
			B2KeyID:          "k1",
			B2ApplicationKey: "a2",
			ResticPassword:   "p3",
		},
	}

	err := client.Backup("/data/backup", []string{"tag1", "tag2"})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	expectedCmd := "RESTIC_REPOSITORY='b2:b:p' B2_ACCOUNT_ID='k1' B2_ACCOUNT_KEY='a2' RESTIC_PASSWORD='p3' restic backup /data/backup --verbose --tag tag1 --tag tag2"
	if executedCmd != expectedCmd {
		t.Errorf("expected last command to be %q, got: %q", expectedCmd, executedCmd)
	}
}

func TestDefaultResticClient_Backup_Success_EmptyTags(t *testing.T) {
	var executedCmd string
	client := &DefaultResticClient{
		commands: &mockCommands{
			execShellCommand: func(cmd string) system.RunnableCommand {
				executedCmd = cmd
				return &mockRunnableCommand{}
			},
		},
		textFormatter: &mockTextFormatter{},
		config: ResticConfig{
			RepositoryURL:    "b2:b:p",
			B2KeyID:          "k1",
			B2ApplicationKey: "a2",
			ResticPassword:   "p3",
		},
	}

	err := client.Backup("/data/backup", []string{})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	expectedCmd := "RESTIC_REPOSITORY='b2:b:p' B2_ACCOUNT_ID='k1' B2_ACCOUNT_KEY='a2' RESTIC_PASSWORD='p3' restic backup /data/backup --verbose"
	if executedCmd != expectedCmd {
		t.Errorf("expected last command to be %q, got: %q", expectedCmd, executedCmd)
	}
}

func TestDefaultResticClient_Backup_Error(t *testing.T) {
	expectedErr := errors.New("backup failed")
	client := &DefaultResticClient{
		commands: &mockCommands{
			execShellCommand: func(cmd string) system.RunnableCommand {
				return &mockRunnableCommand{
					runFunc: func() error {
						return expectedErr
					},
				}
			},
		},
		textFormatter: &mockTextFormatter{},
		config: ResticConfig{
			RepositoryURL:    "b2:b:p",
			B2KeyID:          "k1",
			B2ApplicationKey: "a2",
			ResticPassword:   "p3",
		},
	}

	err := client.Backup("/data/backup", []string{"tag1"})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to be %v, got: %v", expectedErr, err)
	}
}

func TestDefaultResticClient_Forget_Success_WithPrune(t *testing.T) {
	var executedCmd string
	client := &DefaultResticClient{
		commands: &mockCommands{
			execShellCommand: func(cmd string) system.RunnableCommand {
				executedCmd = cmd
				return &mockRunnableCommand{}
			},
		},
		textFormatter: &mockTextFormatter{},
		config: ResticConfig{
			RepositoryURL:    "b2:b:p",
			B2KeyID:          "k1",
			B2ApplicationKey: "a2",
			ResticPassword:   "p3",
		},
	}

	err := client.Forget("30d", true)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	expectedCmd := "RESTIC_REPOSITORY='b2:b:p' B2_ACCOUNT_ID='k1' B2_ACCOUNT_KEY='a2' RESTIC_PASSWORD='p3' restic forget --keep-within 30d --prune"
	if executedCmd != expectedCmd {
		t.Errorf("expected last command to be %q, got: %q", expectedCmd, executedCmd)
	}
}

func TestDefaultResticClient_Forget_Success_WithoutPrune(t *testing.T) {
	var executedCmd string
	client := &DefaultResticClient{
		commands: &mockCommands{
			execShellCommand: func(cmd string) system.RunnableCommand {
				executedCmd = cmd
				return &mockRunnableCommand{}
			},
		},
		textFormatter: &mockTextFormatter{},
		config: ResticConfig{
			RepositoryURL:    "b2:b:p",
			B2KeyID:          "k1",
			B2ApplicationKey: "a2",
			ResticPassword:   "p3",
		},
	}

	err := client.Forget("7d", false)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	expectedCmd := "RESTIC_REPOSITORY='b2:b:p' B2_ACCOUNT_ID='k1' B2_ACCOUNT_KEY='a2' RESTIC_PASSWORD='p3' restic forget --keep-within 7d"
	if executedCmd != expectedCmd {
		t.Errorf("expected last command to be %q, got: %q", expectedCmd, executedCmd)
	}
}

func TestDefaultResticClient_Check_Success(t *testing.T) {
	var executedCmd string
	client := &DefaultResticClient{
		commands: &mockCommands{
			execShellCommand: func(cmd string) system.RunnableCommand {
				executedCmd = cmd
				return &mockRunnableCommand{}
			},
		},
		textFormatter: &mockTextFormatter{},
		config: ResticConfig{
			RepositoryURL:    "b2:b:p",
			B2KeyID:          "k1",
			B2ApplicationKey: "a2",
			ResticPassword:   "p3",
		},
	}

	err := client.Check()

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	expectedCmd := "RESTIC_REPOSITORY='b2:b:p' B2_ACCOUNT_ID='k1' B2_ACCOUNT_KEY='a2' RESTIC_PASSWORD='p3' restic check"
	if executedCmd != expectedCmd {
		t.Errorf("expected last command to be %q, got: %q", expectedCmd, executedCmd)
	}
}

func TestDefaultResticClient_Snapshots_Success(t *testing.T) {
	var executedCmd string
	client := &DefaultResticClient{
		commands: &mockCommands{
			execShellCommand: func(cmd string) system.RunnableCommand {
				executedCmd = cmd
				return &mockRunnableCommand{}
			},
		},
		textFormatter: &mockTextFormatter{},
		config: ResticConfig{
			RepositoryURL:    "b2:b:p",
			B2KeyID:          "k1",
			B2ApplicationKey: "a2",
			ResticPassword:   "p3",
		},
	}

	err := client.Snapshots()

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	expectedCmd := "RESTIC_REPOSITORY='b2:b:p' B2_ACCOUNT_ID='k1' B2_ACCOUNT_KEY='a2' RESTIC_PASSWORD='p3' restic snapshots"
	if executedCmd != expectedCmd {
		t.Errorf("expected last command to be %q, got: %q", expectedCmd, executedCmd)
	}
}

func TestDefaultResticClient_ListFiles_Success(t *testing.T) {
	var executedCmd string
	client := &DefaultResticClient{
		commands: &mockCommands{
			execShellCommand: func(cmd string) system.RunnableCommand {
				executedCmd = cmd
				return &mockRunnableCommand{}
			},
		},
		textFormatter: &mockTextFormatter{},
		config: ResticConfig{
			RepositoryURL:    "b2:b:p",
			B2KeyID:          "k1",
			B2ApplicationKey: "a2",
			ResticPassword:   "p3",
		},
	}

	err := client.ListFiles("abc123")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	expectedCmd := "RESTIC_REPOSITORY='b2:b:p' B2_ACCOUNT_ID='k1' B2_ACCOUNT_KEY='a2' RESTIC_PASSWORD='p3' restic ls abc123"
	if executedCmd != expectedCmd {
		t.Errorf("expected last command to be %q, got: %q", expectedCmd, executedCmd)
	}
}

func TestDefaultResticClient_Restore_Success(t *testing.T) {
	var executedCmd string
	client := &DefaultResticClient{
		commands: &mockCommands{
			execShellCommand: func(cmd string) system.RunnableCommand {
				executedCmd = cmd
				return &mockRunnableCommand{}
			},
		},
		textFormatter: &mockTextFormatter{},
		config: ResticConfig{
			RepositoryURL:    "b2:b:p",
			B2KeyID:          "k1",
			B2ApplicationKey: "a2",
			ResticPassword:   "p3",
		},
	}

	err := client.Restore("/restore/path")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	expectedCmd := "RESTIC_REPOSITORY='b2:b:p' B2_ACCOUNT_ID='k1' B2_ACCOUNT_KEY='a2' RESTIC_PASSWORD='p3' restic restore latest --target '/restore/path' --verbose"
	if executedCmd != expectedCmd {
		t.Errorf("expected last command to be %q, got: %q", expectedCmd, executedCmd)
	}
}
