#!/usr/bin/env sh
set -eu

##### PAPERLESS RESTORATION SCRIPT
# See https://docs.paperless-ngx.com/administration/#importer
#####

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
# Clean export directory and copy restored files
rm -rf "${HOMELAB_PAPERLESS_WEB_EXPORT_PATH:?}"/*
cp -r "$RESTORED_HOMELAB_PAPERLESS_DATA"/* "$HOMELAB_PAPERLESS_WEB_EXPORT_PATH"
# Start required services and run importer
docker compose up -d paperless-redis paperless-db paperless
sleep 30
docker compose exec -T paperless document_importer ../export
