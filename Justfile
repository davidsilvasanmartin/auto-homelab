set dotenv-load := true

uv := "uv"
docker := "docker"

# [ HELP] List available commands. Gets executed when running `just` with no args
default:
    @just --list --unsorted

# [🔧 APP] Interactive script that creates a `.env.<timestamp>` file
configure:
    {{uv}} run --env-file=.env -m scripts.configure

# [🔧 APP] Starts all services
start:
    {{docker}} compose up -d

# [🔧 APP] Creates a local backup of all services' data
backup-local:
    {{uv}} run --env-file=.env -m scripts.backup

# [🧪 DEV] Runs the tests of Python scripts
dev-test:
    {{uv}} run --env-file=.env -m pytest scripts/tests

# TODO commands for bootstrapping the Python project? As in, installing dependencies for the first time. uv something ??

# [🧪 DEV] Add development dependencies with uv. Example: `just add-dev "black>=24.8,<25" isort mypy`
dev-add-dev +pkgs:
    {{uv}} add --dev {{pkgs}}

# [🧪 DEV] Lint the whole project and auto-fix what Ruff safely can
dev-lint:
    {{uv}} run ruff check . --fix

# [🧪 DEV] Checks that files are formatted correctly
dev-fmt-check:
    {{uv}} run ruff format . --check

# [🧪 DEV] Format all files
dev-fmt:
    {{uv}} run ruff format .

# [🧪 DEV] Explains a linting rule. Example: `just dev-explain F401`
dev-explain linting-rule:
    {{uv}} run ruff rule {{linting-rule}}
