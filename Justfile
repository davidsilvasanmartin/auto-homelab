set dotenv-load := true

uv := "uv"
docker := "docker"

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
    # TODO !!
