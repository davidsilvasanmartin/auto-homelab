set dotenv-load := true

uv := "uv"
docker := "docker"

# TODO fix command "just", which runs configure
# TODO add comments everywhere

configure:
    echo "Configuring app..."
    {{uv}} run --env-file=.env -m scripts.configure

up:
    echo "Enabling all services..."
    {{docker}} compose up -d

backup:
    echo "Creating a local backup of the data..."
    {{uv}} run --env-file=.env -m scripts.backup

test:
    echo "Running all tests..."
    {{uv}} run --env-file=.env -m pytest scripts/tests

# Add development dependencies with uv. Example:  just add-dev "black>=24.8,<25" isort mypy
add-dev +pkgs:
    {{uv}} add --dev {{pkgs}}
