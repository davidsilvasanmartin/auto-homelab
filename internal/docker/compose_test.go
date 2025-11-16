package docker

import (
	"fmt"
	"os"
	"testing"
)

func TestBuildDockerComposeCommandStr_FormatIsCorrect(t *testing.T) {
	cmd := "up -d --build service1 service2"

	result := BuildDockerComposeCommandStr(cmd)

	expectedUID := os.Getuid()
	expectedGID := os.Getgid()
	expectedCmd := fmt.Sprintf("HOMELAB_GENERAL_UID=%d HOMELAB_GENERAL_GID=%d docker compose %s", expectedUID, expectedGID, cmd)
	if result != expectedCmd {
		t.Errorf("expected command %q, got %q", expectedCmd, result)
	}
}

func TestBuildDockerComposeCommandStr_EmptyCommand(t *testing.T) {
	cmd := ""

	result := BuildDockerComposeCommandStr(cmd)

	uid := os.Getuid()
	gid := os.Getgid()
	expectedCmd := fmt.Sprintf("HOMELAB_GENERAL_UID=%d HOMELAB_GENERAL_GID=%d docker compose ", uid, gid)
	if result != expectedCmd {
		t.Errorf("expected command %q, got %q", expectedCmd, result)
	}
}
