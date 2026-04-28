# Kubernetes Migration: Backup Strategy

This document covers how the current two-phase backup (local copy → restic cloud upload)
maps to Kubernetes, what improves, and what the recommended architecture looks like.

---

## The current backup problem

The current workflow has a known pain point: `backup local` must create a full local copy of
all service data on the host before `backup cloud` can upload anything to Backblaze B2.
This means you need enough free disk space for a complete second copy of all data. For
services like Immich (which may hold hundreds of gigabytes of photos), this is expensive.

The root cause is that `restic backup` is called from the host and given the path to the
local copy directory — it cannot access the containers' internal volumes directly.

Kubernetes does not fully solve the disk-doubling problem for database dumps (you still need
to write the SQL somewhere before uploading it), but it significantly improves the file
backup story by allowing a backup pod to mount the same PVC as the service being backed up.

---

## Approach 1: CronJob with shared PVC (recommended for file data)

For services that store data in PVCs (Immich uploads, Calibre library, Navidrome data,
Paperless media), a backup `CronJob` can mount the same PVC as the service and run
`restic backup` directly, without any intermediate local copy.

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: backup-immich-uploads
  namespace: homelab
spec:
  schedule: "0 2 * * *"   # 2 AM daily
  concurrencyPolicy: Forbid
  jobTemplate:
    spec:
      template:
        spec:
          restartPolicy: OnFailure
          containers:
            - name: restic
              image: restic/restic:0.17.3
              command:
                - /bin/sh
                - -c
                - |
                  restic snapshots || restic init
                  restic backup /data --tag immich-uploads --tag automatic
                  restic forget --keep-within ${RESTIC_RETENTION_DAYS}d --prune
              env:
                - name: RESTIC_REPOSITORY
                  valueFrom:
                    secretKeyRef:
                      name: backup-credentials
                      key: RESTIC_REPOSITORY
                - name: RESTIC_PASSWORD
                  valueFrom:
                    secretKeyRef:
                      name: backup-credentials
                      key: RESTIC_PASSWORD
                - name: B2_ACCOUNT_ID
                  valueFrom:
                    secretKeyRef:
                      name: backup-credentials
                      key: B2_ACCOUNT_ID
                - name: B2_ACCOUNT_KEY
                  valueFrom:
                    secretKeyRef:
                      name: backup-credentials
                      key: B2_ACCOUNT_KEY
                - name: RESTIC_RETENTION_DAYS
                  valueFrom:
                    configMapKeyRef:
                      name: backup-config
                      key: RETENTION_DAYS
              volumeMounts:
                - name: immich-uploads
                  mountPath: /data
                  readOnly: true    # backup pod only reads
          volumes:
            - name: immich-uploads
              persistentVolumeClaim:
                claimName: immich-uploads
                readOnly: true
```

**What this eliminates**: No intermediate local copy needed. Restic reads directly from the
PVC and streams to Backblaze B2. Disk space requirement drops to near zero (restic itself
uses some temp space, but nothing proportional to the dataset).

**Caveat**: The Immich application continues running while the backup pod reads its PVC.
For a `ReadWriteOnce` PVC on a single node this is fine — both pods can read the same
volume simultaneously. Writes from Immich during backup may produce a slightly inconsistent
snapshot for new uploads added mid-backup, which is acceptable for a photo library.

---

## Approach 2: Database dump CronJob

Databases (PostgreSQL for Paperless and Immich, MariaDB for Firefly) cannot be backed up
safely by copying raw data files while the database is running. The existing `pg_dump` /
`mariadb-dump` approach is correct and stays in Kubernetes — it just runs in a CronJob
that uses `kubectl exec` (the `PodExec` subresource) to dump into the backup pod's storage.

Alternatively, the dump can run as a sidecar that shares an `emptyDir` with a restic
container in the same pod:

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: backup-immich-db
  namespace: homelab
spec:
  schedule: "30 1 * * *"   # 1:30 AM daily, before the file backup at 2 AM
  concurrencyPolicy: Forbid
  jobTemplate:
    spec:
      template:
        spec:
          restartPolicy: OnFailure
          initContainers:
            - name: pg-dump
              image: postgres:14
              command:
                - /bin/sh
                - -c
                - |
                  pg_dump -h immich-db -U ${POSTGRES_USER} ${POSTGRES_DB} \
                    > /dump/database.sql
              env:
                - name: PGPASSWORD
                  valueFrom:
                    secretKeyRef:
                      name: immich-db-secret
                      key: POSTGRES_PASSWORD
                - name: POSTGRES_USER
                  valueFrom:
                    secretKeyRef:
                      name: immich-db-secret
                      key: POSTGRES_USER
                - name: POSTGRES_DB
                  valueFrom:
                    secretKeyRef:
                      name: immich-db-secret
                      key: POSTGRES_DB
              volumeMounts:
                - name: dump
                  mountPath: /dump
          containers:
            - name: restic
              image: restic/restic:0.17.3
              command:
                - /bin/sh
                - -c
                - |
                  restic snapshots || restic init
                  restic backup /dump --tag immich-db --tag automatic
                  restic forget --keep-within ${RESTIC_RETENTION_DAYS}d --prune
              env:
                # ... same backup-credentials as above
              volumeMounts:
                - name: dump
                  mountPath: /dump
          volumes:
            - name: dump
              emptyDir: {}    # shared between initContainer and main container; ephemeral
```

The `emptyDir` volume is automatically destroyed when the job pod terminates. The SQL dump
exists only long enough to be uploaded to restic. **No permanent local copy needed.**

---

## Paperless: export before backup

The current `DirectoryLocalBackup` for Paperless runs a pre-command:

```
docker compose exec paperless document_exporter -d ../export
```

In Kubernetes this becomes a `CronJob` with an `initContainer` that runs the exporter:

```yaml
initContainers:
  - name: paperless-export
    image: paperlessngx/paperless-ngx:2.16.1
    command: ["python3", "manage.py", "document_exporter", "/export"]
    env:
      # ... same env vars as the paperless deployment
    volumeMounts:
      - name: export
        mountPath: /export
      # mount paperless-data, paperless-media etc. too — the exporter needs them
```

Followed by a restic container that backs up the `/export` volume to B2.

---

## Complete backup schedule

| CronJob name | Schedule | What it backs up |
|---|---|---|
| `backup-immich-db` | 1:30 AM daily | Immich PostgreSQL (pg_dump → restic) |
| `backup-paperless-db` | 1:35 AM daily | Paperless PostgreSQL (pg_dump → restic) |
| `backup-firefly-db` | 1:40 AM daily | Firefly MariaDB (mariadb-dump → restic) |
| `backup-paperless-export` | 1:50 AM daily | Paperless document export → restic |
| `backup-immich-uploads` | 2:00 AM daily | Immich uploads PVC → restic |
| `backup-calibre` | 2:30 AM daily | Calibre library + conf PVCs → restic |
| `backup-navidrome` | 3:00 AM daily | Navidrome data PVC → restic |
| `backup-firefly-uploads` | 3:05 AM daily | Firefly uploads PVC → restic |

Use `concurrencyPolicy: Forbid` on all CronJobs to prevent overlapping runs.

---

## Retaining the Go CLI backup commands

The Go CLI's `backup local` and `backup cloud` commands can be retained as manual triggers
for the same operations — useful when you want to take a backup outside the schedule.

The implementation changes:
- `backup local` → creates a one-off `Job` from the same spec as the relevant CronJob
  (using `client-go` to create a `Job` in the `homelab` namespace), or directly runs
  the dump+restic pipeline locally if the CLI has access to `kubectl exec`
- `backup cloud list` → remains a `restic snapshots` shell-out; no change needed
- `backup cloud restore <dir>` → remains a `restic restore` shell-out; no change needed

---

## Restoring from backup

Restoration in Kubernetes is the same process as today, but with pod exec instead of
docker exec:

### Database restoration

```bash
# 1. Scale the app down to avoid writes during restore
kubectl scale deployment immich --replicas=0 -n homelab

# 2. Run a restore pod that has the DB secret and can reach immich-db
kubectl run restore-immich-db --rm -it \
  --namespace homelab \
  --image=postgres:14 \
  --env="PGPASSWORD=<password>" \
  -- psql -h immich-db -U immich immich < /path/to/database.sql
# In practice: use a Job that mounts a PVC containing the restored SQL file,
# or restore from restic first, then pipe from file

# 3. Scale the app back up
kubectl scale deployment immich --replicas=1 -n homelab
```

### File restoration

```bash
# Run a restic restore job that writes directly to the immich-uploads PVC
kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: restore-immich-uploads
  namespace: homelab
spec:
  template:
    spec:
      restartPolicy: Never
      containers:
        - name: restic
          image: restic/restic:0.17.3
          command: ["restic", "restore", "latest", "--target", "/data"]
          env:
            # ... backup-credentials secret
          volumeMounts:
            - name: immich-uploads
              mountPath: /data
      volumes:
        - name: immich-uploads
          persistentVolumeClaim:
            claimName: immich-uploads
EOF
```

---

## What the migration fixes about backup

| Problem today | Kubernetes solution |
|---|---|
| Local copy needs 2× disk space | CronJob reads PVC directly, emptyDir for DB dumps |
| Manual invocation required | CronJobs run on schedule automatically |
| Docker context must be correct | CronJobs run in-cluster, no context management |
| Backup fails silently | CronJob failures show as Failed Job objects; alert with `kubectl get jobs` |
| Restore scripts are shell-only | Go CLI can create restore Jobs via client-go |
| `backup local` must run before `backup cloud` | Two-phase is eliminated; restic runs directly from PVC |
