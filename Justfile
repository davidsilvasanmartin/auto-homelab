set dotenv-load := true

uv := "uv"
docker := "docker"

# [ HELP] List available commands. Gets executed when running `just` with no args
default:
    @just --list --unsorted

# [🔧 APP] Interactive script that creates a `.env.<timestamp>` file
configure:
    {{uv}} run --env-file=.env -m scripts.configuration.configure

# [🔧 APP] Starts a service, or all services if one is not specified. Example: `just start` // `just start calibre`
start service="":
    if [ -n "{{service}}" ]; then {{docker}} compose up -d {{service}}; else {{docker}} compose up -d; fi

# [🔧 APP] Stops a service, or all services if one is not specified. Example: `just stop` // `just stop calibre`
stop service="":
    if [ -n "{{service}}" ]; then {{docker}} compose stop {{service}}; else {{docker}} compose stop; fi

# [🔧 APP] Creates a local backup of all services' data
backup-local:
    {{uv}} run --env-file=.env -m scripts.backup.local

# [🔧 APP] Syncs the local backup to the cloud. The `backup-local` must be ran first
backup-cloud:
    {{uv}} run --env-file=.env -m scripts.cloud_backup

# [🔧 APP] Lists the backup snapshots that exist on the configured cloud bucket
backup-cloud-list:
    {{uv}} run --env-file=.env -m scripts.cloud_backup --command=list

# [🔧 APP] Restores the paperless-ngx data
restore-paperless:
    ./scripts-shell/restore_paperless.sh

# [🔧 APP] Restores the immich data
restore-immich:
    ./scripts-shell/restore_immich.sh

# [🧪 DEV] Runs the tests of Python scripts
dev-test:
    {{uv}} run --env-file=.env -m pytest scripts/tests

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
    {{uv}} run ruff format .

# [🧪 DEV] Explains a linting rule. Example: `just dev-explain F401`
dev-explain linting-rule:
    {{uv}} run ruff rule {{linting-rule}}

# [📊 STAT] Count lines of code (optional dir argument, default "."). Example: `just stat-loc` or `just stat-loc scripts`
stat-loc dir=".":
    {{uv}} run -m scripts.loc {{dir}}
