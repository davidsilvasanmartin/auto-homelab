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
test:
    {{uv}} run --env-file=.env -m pytest scripts/tests

# [🧪 DEV] Add development dependencies with uv. Example: `just add-dev "black>=24.8,<25" isort mypy`
add-dev +pkgs:
    {{uv}} add --dev {{pkgs}}
