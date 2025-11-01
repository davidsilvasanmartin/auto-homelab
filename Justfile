set dotenv-load := true

uv := "uv"
docker := "docker"

# [ HELP] List available commands. Gets executed when running `just` with no args
default:
    @just --list --unsorted

# [ HELP] Show help of the main script
help:
    go run . -h

# [🔧 APP] Interactive script that creates a `.env.<timestamp>` file
configure:
    if [  -f ".env" ]; then {{uv}} run --env-file=.env -m scripts.configuration.configure; else {{uv}} run -m scripts.configuration.configure; fi

# [🔧 APP] Starts a service, or all services if one is not specified. Example: `just start` // `just start calibre`
start *services="":
    go run . --log-level debug start {{services}}

# [🔧 APP] Stops a service, or all services if one is not specified. Example: `just stop` // `just stop calibre`
stop *services="":
    go run . --log-level debug stop {{services}}

# [🔧 APP] Creates a local backup of all services' data
backup-local:
    go run . --log-level debug backup local

# [🔧 APP] Syncs the local backup to the cloud. The `backup-local` must be ran first
backup-cloud:
    {{uv}} run --env-file=.env -m scripts.backup.cloud --command=backup

# [🔧 APP] Lists the backup snapshots that exist on the configured cloud bucket
backup-cloud-list:
    {{uv}} run --env-file=.env -m scripts.backup.cloud --command=list

# [🔧 APP] Restores the paperless-ngx data
restore-paperless:
    ./scripts-shell/restore_paperless.sh

# [🔧 APP] Restores the immich data
restore-immich:
    ./scripts-shell/restore_immich.sh

# [🔧 APP] Fixes the permissions of directories used by the app (Mac or Linux)
fix-perms:
    UID_CURR=$(id -u); \
    GID_CURR=$(id -g); \
    echo ${UID_CURR}; \
    echo ${GID_CURR};

# [🧪 DEV] Runs the tests of Go scripts
dev-test:
    go test ./...

# [🧪 DEV] Runs the tests of Go scripts, ignoring cache
dev-test-no-cache:
    go test ./... -count=1

# TODO commands for bootstrapping the Python project? As in, installing dependencies for the first time. uv something ??
# [🧪 DEV] Add dependencies with uv. Example: `just dev-add "requests>=24.8,<25" pandas`
dev-add +pkgs:
    {{uv}} add {{pkgs}}

# [🧪 DEV] Add development dependencies with uv. Example: `just dev-add-dev "black>=24.8,<25" isort mypy`
dev-add-dev +pkgs:
    {{uv}} add --dev {{pkgs}}

# [🧪 DEV] Lint the whole project and auto-fix what Ruff safely can
dev-lint:
    {{uv}} run ruff check . --fix

# [🧪 DEV] Checks that files are formatted correctly
dev-check-fmt:
    {{uv}} run ruff format . --check

# [🧪 DEV] Checks that all Python code has the correct types
dev-check-types:
    {{uv}} run pyright .

# [🧪 DEV] Format all files
dev-fmt:
    go fmt ./...

# [🧪 DEV] Explains a linting rule. Example: `just dev-explain F401`
dev-explain linting-rule:
    {{uv}} run ruff rule {{linting-rule}}

# [📊 STAT] Count lines of code (optional dir argument, default "."). Example: `just stat-loc` or `just stat-loc scripts`
stat-loc dir=".":
    {{uv}} run -m scripts.loc {{dir}}
