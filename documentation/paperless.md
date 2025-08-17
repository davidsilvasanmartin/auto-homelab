# Paperless-ngx

## Backup

To back up the files and the metadata, run
```shell
 docker compose exec -T paperless-ngx-webserver document_exporter -d ../export
```
- Add the `-z` option if a single zip file is needed.
- Please note, the `../export` path is a path inside the Docker container and must
be left unchanged.

### Backup documentation
- https://docs.paperless-ngx.com/administration/#backup
- https://docs.paperless-ngx.com/administration/#exporter

