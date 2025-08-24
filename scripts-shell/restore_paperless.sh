#!/usr/bin/env sh
set -eu

##### PAPERLESS RESTORATION SCRIPT
# See https://docs.paperless-ngx.com/administration/#importer
#####

# WARNING AND CONFIRMATION
echo "WARNING: This operation will DELETE Paperless-ngx containers, volumes, and all existing data." 1>&2
echo "Type 'yes' to confirm and continue, or anything else to abort." 1>&2
printf "Confirm (type 'yes'): " 1>&2
if [ -r /dev/tty ]; then
    IFS= read -r __CONFIRM </dev/tty
else
    IFS= read -r __CONFIRM
fi
if [ "$__CONFIRM" != "yes" ]; then
    echo "Aborted by user." 1>&2
    exit 1
fi

# Ensure .env exists
if [ ! -f .env ]; then
    echo "Error: .env not found. Please create it before running this command."
    exit 1
fi
# Prompt for the restored data path
printf "Enter path to your restored Paperless-ngx data (export): " 1>&2
if [ -r /dev/tty ]; then
    IFS= read -r RESTORED_HOMELAB_PAPERLESS_DATA </dev/tty
else
    IFS= read -r RESTORED_HOMELAB_PAPERLESS_DATA
fi
if [ -z "$RESTORED_HOMELAB_PAPERLESS_DATA" ] || [ ! -d "$RESTORED_HOMELAB_PAPERLESS_DATA" ]; then
    echo "Error: Invalid path: $RESTORED_HOMELAB_PAPERLESS_DATA"
    exit 1
fi

### Remove everything
rm -rf "${HOMELAB_PAPERLESS_REDIS_DATA_PATH:?}"/*
rm -rf "${HOMELAB_PAPERLESS_DB_DATA_PATH:?}"/*
rm -rf "${HOMELAB_PAPERLESS_WEB_DATA_PATH:?}"/*
rm -rf "${HOMELAB_PAPERLESS_WEB_MEDIA_PATH:?}"/*
rm -rf "${HOMELAB_PAPERLESS_WEB_EXPORT_PATH:?}"/*
rm -rf "${HOMELAB_PAPERLESS_WEB_CONSUME_PATH:?}"/*

### Restore the data and start up the services
cp -r "$RESTORED_HOMELAB_PAPERLESS_DATA"/* "$HOMELAB_PAPERLESS_WEB_EXPORT_PATH"
# Completely remove the service containers and volumes
docker compose down -v paperless-redis paperless-db paperless
# Start required services and run importer
docker compose up -d paperless-redis paperless-db
sleep 30
docker compose exec -T paperless document_importer ../export
docker compose up -d paperless
