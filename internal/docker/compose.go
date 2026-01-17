package docker

import (
	"fmt"
	"os"
	"strings"
)

// BuildDockerComposeCommandStr builds a Docker Compose command in a way that's safe to use. For example, it sets the
// environment variables that specify which user and group are running the command. The resulting command is
// `docker compose ${cmd}`, where `${cmd} is the value of this function's `cmd` input argument.
// If dockerContext is non-empty, the command will use that Docker context.
func BuildDockerComposeCommandStr(cmd string, dockerContext string) string {
	uid := os.Getuid()
	gid := os.Getgid()

	var cmdParts []string
	cmdParts = append(cmdParts, fmt.Sprintf("HOMELAB_GENERAL_UID=%d", uid))
	cmdParts = append(cmdParts, fmt.Sprintf("HOMELAB_GENERAL_GID=%d", gid))

	if dockerContext != "" {
		cmdParts = append(cmdParts, fmt.Sprintf("docker --context %s compose", dockerContext))
	} else {
		cmdParts = append(cmdParts, "docker compose")
	}

	cmdParts = append(cmdParts, cmd)

	return strings.Join(cmdParts, " ")
}
