#!/usr/bin/env sh
set -eu

##### IMMICH RESTORATION SCRIPT
# See https://immich.app/docs/administration/backup-and-restore/
#####

# WARNING AND CONFIRMATION
echo "NOTE: For some reason Docker changes the permissions of some directories, so it is possible"
echo "that this script fails due to that. If this happens, set yourself manually (chown) as the owner"
echo "of the problematic files"
echo ""
echo "WARNING: This operation will DELETE Immich containers, volumes, and all existing data." 1>&2
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

# Prompt for the restored files
printf "Enter path to your restored Immich database .sql file: " 1>&2
if [ -r /dev/tty ]; then
    IFS= read -r RESTORED_HOMELAB_IMMICH_DB </dev/tty
else
    IFS= read -r RESTORED_HOMELAB_IMMICH_DB
fi
if [ -z "$RESTORED_HOMELAB_IMMICH_DB" ] || [ ! -f "$RESTORED_HOMELAB_IMMICH_DB" ]; then
    echo "Error: Invalid path: $RESTORED_HOMELAB_IMMICH_DB"
    exit 1
fi
case "$RESTORED_HOMELAB_IMMICH_DB" in
    *.sql) : ;;
    *)
        echo "Error: Database dump must be a .sql file: $RESTORED_HOMELAB_IMMICH_DB" 1>&2
        exit 1
        ;;
esac

printf "Enter path to your restored Immich \"upload\" directory: " 1>&2
if [ -r /dev/tty ]; then
    IFS= read -r RESTORED_HOMELAB_IMMICH_UPLOAD </dev/tty
else
    IFS= read -r RESTORED_HOMELAB_IMMICH_UPLOAD
fi
if [ -z "$RESTORED_HOMELAB_IMMICH_UPLOAD" ] || [ ! -d "$RESTORED_HOMELAB_IMMICH_UPLOAD" ]; then
    echo "Error: Invalid path: $RESTORED_HOMELAB_IMMICH_UPLOAD"
    exit 1
fi

### Remove everything
# Completely remove the service containers and volumes before performing the restoration
docker compose down -v immich-redis immich-machine-learning immich-db immich
# Completely remove anything inside all directories used by immich. These directories
# should exist because they were created with the initial configuration script
rm -rf "${HOMELAB_IMMICH_WEB_UPLOAD_PATH:?}"/*
rm -rf "${HOMELAB_IMMICH_DB_DATA_PATH:?}"/*
rm -rf "${HOMELAB_IMMICH_ML_CACHE_DATA_PATH:?}"/*
rm -rf "${HOMELAB_IMMICH_REDIS_DATA_PATH:?}"/*

### Restore the files
mkdir -p "${HOMELAB_IMMICH_WEB_UPLOAD_PATH:?}"
cp -R "$RESTORED_HOMELAB_IMMICH_UPLOAD"/. "$HOMELAB_IMMICH_WEB_UPLOAD_PATH"/

### Restore the database and start up the services
docker compose up -d immich-redis immich-machine-learning immich-db
# Give the database a moment to be ready to accept connections
sleep 30
# Restore using psql inside the database container
docker exec -i "${HOMELAB_IMMICH_DB_CONTAINER_NAME}" /bin/bash -c \
    "PGPASSWORD='${HOMELAB_IMMICH_DB_PASSWORD}' psql --username='${HOMELAB_IMMICH_DB_USER}' --dbname='${HOMELAB_IMMICH_DB_DATABASE}'" \
    < "$RESTORED_HOMELAB_IMMICH_DB"
docker compose up -d immich
