# Auto-homelab

## Prerequisites

- [Docker](https://www.docker.com/get-started/)
- [Just](https://github.com/casey/just)
- [uv](https://github.com/astral-sh/uv)

## How to use

You can get a list of all the available commands, along with a small explanation, by running:
```shell
just
```

- Commands whose documentation starts with `[🔧APP]` provide the functionalities of this project.
- Commands whose documentation starts with `[🧪DEV]` are for development purposes only. You should
not use them unless you are contributing to this project.

TODO... This section is under construction ...

## Project structure

- `Justfile`: contains all the commands that are used to run the app, and to develop new features for the app
- `docker-compose.yml`: contains the definition of all the services the homelab is composed of
- `pyproject.toml`, `uv.lock`, `.python-version`: these files are used by `uv` to manage the Python environment
- `documentation/`: in this directory you will find several articles documenting different aspects of the project. Some of
these are for my own reference, so that I can remember how I did something.
- `files/`: contains configuration files and scripts for various services
- `scripts/`: contains Python scripts that give this project some of its functionality
- `scripts-shell/`: contains shell scripts that provide some extra functionalities
- `scripts/env.config.json`: this file contains the schema of the environment variables that are used by the app

## Disaster recovery: How to restore the services once we have restored the backup files

### Important notes

#### ALL EXISTING DATA WILL BE LOST
Recovery scripts assume that you want to destroy a service instance and rebuild it from the ground up.
Therefore, they will remove all existing data before restoring the backup files.

#### Directory permissions
For some reason Docker changes the ownership and permissions of some directories,
so it is possible that this script fails due to that. If this happens, set yourself manually (chown)
as the owner of the problematic files

### Restoring databases in general

To see an example of how to restore a Postgres database, [see the official Immich
docs](https://immich.app/docs/administration/backup-and-restore#manual-backup-and-restore).

It is possible that database restoration processes fail if the name of the old database is different from the new one.
That's why it's recommended to leave database names untouched. They are defined in the `env.config.json` file.

### Restoring Calibre

Just copy the files over. Navigate to this project's root directory and run:

```shell
source .env
export RESTORED_HOMELAB_CALIBRE_CONF_PATH=<path_to_your_restored_calibre_config>
export RESTORED_HOMELAB_CALIBRE_LIBRARY_PATH=<path_to_your_restored_calibre_library>
cp -r $RESTORED_HOMELAB_CALIBRE_CONF_PATH/* $HOMELAB_CALIBRE_CONF_PATH
cp -r $RESTORED_HOMELAB_CALIBRE_LIBRARY_PATH/* $HOMELAB_CALIBRE_LIBRARY_PATH
```

You will need to replace the placeholders above (`<path_to...>`) with the correct values.

### Restoring Firefly III

1. Make sure the Firefly database container is not running
2. Run the following command:

```shell
source .env
export RESTORED_HOMELAB_FIREFLY_DB_SQL=<path_to_your_restored_firefly_db_sql>
docker compose up -d --no-deps firefly-db
sleep 20 
docker exec -i "${HOMELAB_FIREFLY_DB_CONTAINER_NAME}" mariadb -u"${HOMELAB_FIREFLY_DB_USER}" -p"${HOMELAB_FIREFLY_DB_PASSWORD}" "${HOMELAB_FIREFLY_DB_DATABASE}" < "${RESTORED_HOMELAB_FIREFLY_DB_SQL}"
```

You will need to replace the placeholder shown above (`<path_to...>`). It should contain the absolute path to the SQL file 
that contains the restored Firefly III database.

#### Notes

- The service is started by its name: `firefly-db`. This name is hardcoded in the `docker-compose.yml` file, it's not the name of the container.
- The `sleep 15` command is needed because the database takes a while to start up.

### Restoring Paperless-ngx

The backup uses the [document exporter](https://docs.paperless-ngx.com/administration/#exporter) tool,
hence the recovery script uses the [document importer](https://docs.paperless-ngx.com/administration/#importer) tool.

The restoration is performed by running the following command:

```shell
just restore-paperless
```

Notes:
- Ensure a valid `.env` file exists in the project root before running the command.
- You will be prompted for the path to your restored Paperless-ngx export data.
- If you performed the backup using this project, the backed-up files will contain
the username and password for the Paperless-ngx account. Use these credentias to
log in after the restoration is complete.

### Restoring Immich

See [Immich docs on restoring data](https://immich.app/docs/administration/backup-and-restore/).

The restoration is performed by running the following command:

```shell
just restore-immich
```
