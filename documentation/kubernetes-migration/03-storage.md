# Kubernetes Migration: Storage

This document covers how Docker bind mounts and named volumes translate to Kubernetes
`PersistentVolume` and `PersistentVolumeClaim` objects, and how to set up a storage class
that works on a single-node Debian host.

---

## The current model

All data lives on the host filesystem. Paths are configured via the `.env` file:

```bash
HOMELAB_PAPERLESS_WEB_DATA_PATH=/mnt/data/paperless/data
HOMELAB_IMMICH_WEB_UPLOAD_PATH=/mnt/data/immich/uploads
HOMELAB_FIREFLY_DB_DATA_PATH=/mnt/data/firefly/db
```

Docker bind-mounts these paths directly into containers. Simple, transparent, and easy to
back up — you know exactly where the data is.

---

## The Kubernetes storage model

Kubernetes introduces a three-layer abstraction:

```
StorageClass  →  PersistentVolume  →  PersistentVolumeClaim  →  Pod
```

- **StorageClass**: Describes how storage is provisioned (manually or dynamically). Think of
  it as a template for PVs.
- **PersistentVolume (PV)**: A piece of storage in the cluster. On a single node it is a
  directory on the host filesystem.
- **PersistentVolumeClaim (PVC)**: A request for storage by a pod. The claim binds to a PV.

For a single-node homelab, `local-path-provisioner` (from Rancher) is the best choice. It
dynamically creates PVs in a base directory (e.g. `/opt/local-path-provisioner`) when a PVC
is created.

---

## Install local-path-provisioner

```bash
kubectl apply -f https://raw.githubusercontent.com/rancher/local-path-provisioner/master/deploy/local-path-storage.yaml
```

This installs:
- The provisioner deployment in namespace `local-path-storage`
- A `StorageClass` named `local-path`
- A `ConfigMap` that controls where on the host the directories are created

Make it the default storage class so PVCs that don't specify a class get it automatically:

```bash
kubectl patch storageclass local-path \
  -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'
```

### Customising the base path

By default, local-path-provisioner creates directories under `/opt/local-path-provisioner`.
To keep data on a specific mount (e.g. `/mnt/data`), edit its ConfigMap:

```bash
kubectl edit configmap local-path-config -n local-path-storage
```

Change `paths` in the JSON:

```json
{
  "nodePathMap": [
    {
      "node": "DEFAULT_PATH_FOR_NON_LISTED_NODES",
      "paths": ["/mnt/data/kubernetes"]
    }
  ]
}
```

---

## Defining PersistentVolumeClaims

Each service that needs persistent storage gets one or more PVCs. Example for Paperless:

```yaml
# paperless-storage.yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: paperless-data
  namespace: homelab
spec:
  accessModes:
    - ReadWriteOnce    # single node can mount read-write
  storageClassName: local-path
  resources:
    requests:
      storage: 10Gi
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: paperless-media
  namespace: homelab
spec:
  accessModes:
    - ReadWriteOnce
  storageClassName: local-path
  resources:
    requests:
      storage: 50Gi
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: paperless-export
  namespace: homelab
spec:
  accessModes:
    - ReadWriteOnce
  storageClassName: local-path
  resources:
    requests:
      storage: 5Gi
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: paperless-consume
  namespace: homelab
spec:
  accessModes:
    - ReadWriteOnce
  storageClassName: local-path
  resources:
    requests:
      storage: 1Gi
```

The `storage` value is advisory — `local-path-provisioner` does not enforce quota. Set it
to something meaningful so your intent is documented.

---

## Complete storage map

The table below maps every significant bind mount from `docker-compose.yml` to a PVC name.

| Service | Compose variable | PVC name | Size estimate |
|---|---|---|---|
| AdGuard | `HOMELAB_ADGUARD_CONF_PATH` | `adguard-conf` | 100Mi |
| AdGuard | `HOMELAB_ADGUARD_WORK_PATH` | `adguard-work` | 1Gi |
| Traefik | `HOMELAB_TRAEFIK_CERTS_PATH` | `traefik-certs` | 100Mi |
| Traefik | `HOMELAB_TRAEFIK_LOGS_PATH` | `traefik-logs` | 2Gi |
| Calibre | `HOMELAB_CALIBRE_CONF_PATH` | `calibre-conf` | 1Gi |
| Calibre | `HOMELAB_CALIBRE_INGEST_PATH` | `calibre-ingest` | 5Gi |
| Calibre | `HOMELAB_CALIBRE_LIBRARY_PATH` | `calibre-library` | 50Gi |
| Paperless-redis | `HOMELAB_PAPERLESS_REDIS_DATA_PATH` | `paperless-redis-data` | 500Mi |
| Paperless-db | `HOMELAB_PAPERLESS_DB_DATA_PATH` | `paperless-db-data` | 5Gi |
| Paperless | `HOMELAB_PAPERLESS_WEB_DATA_PATH` | `paperless-data` | 10Gi |
| Paperless | `HOMELAB_PAPERLESS_WEB_MEDIA_PATH` | `paperless-media` | 50Gi |
| Paperless | `HOMELAB_PAPERLESS_WEB_EXPORT_PATH` | `paperless-export` | 5Gi |
| Paperless | `HOMELAB_PAPERLESS_WEB_CONSUME_PATH` | `paperless-consume` | 1Gi |
| Immich | `HOMELAB_IMMICH_WEB_UPLOAD_PATH` | `immich-uploads` | 500Gi |
| Immich ML | `HOMELAB_IMMICH_ML_CACHE_DATA_PATH` | `immich-ml-cache` | 20Gi |
| Immich-redis | `HOMELAB_IMMICH_REDIS_DATA_PATH` | `immich-redis-data` | 500Mi |
| Immich-db | `HOMELAB_IMMICH_DB_DATA_PATH` | `immich-db-data` | 20Gi |
| Portainer | `HOMELAB_PORTAINER_DATA_PATH` | `portainer-data` | 1Gi |
| Firefly | `HOMELAB_FIREFLY_UPLOAD_PATH` | `firefly-uploads` | 5Gi |
| Firefly-db | `HOMELAB_FIREFLY_DB_DATA_PATH` | `firefly-db-data` | 10Gi |
| Navidrome | `HOMELAB_NAVIDROME_DATA_PATH` | `navidrome-data` | 2Gi |
| Navidrome | `HOMELAB_NAVIDROME_MUSIC_PATH` | `navidrome-music` | (your library) |

---

## Read-only mounts

Some volumes are read-only in the compose file (`:ro`). In Kubernetes, use a separate volume
entry with `readOnly: true`:

```yaml
# docker-compose.yml
- ./files/adguard/scripts:/auto-homelab/scripts:ro

# Kubernetes equivalent:
volumes:
  - name: adguard-scripts
    configMap:
      name: adguard-scripts   # scripts stored in a ConfigMap
volumeMounts:
  - name: adguard-scripts
    mountPath: /auto-homelab/scripts
    readOnly: true
```

Shell scripts and config files that are currently in `./files/` should be stored in
`ConfigMap` objects. This keeps them under Git version control (they already are) and makes
them available to pods without bind-mounting from the host.

---

## Migrating existing data

When you migrate a service, you need to move existing data from the Docker bind-mount path
to the path that `local-path-provisioner` will use. The cleanest approach:

1. Stop the service in Docker Compose.
2. Apply the PVC manifest — local-path-provisioner creates the backing directory.
3. Find the backing directory:
   ```bash
   kubectl get pv <pv-name> -o jsonpath='{.spec.local.path}'
   ```
4. Copy data from the old bind-mount path to the new PV path:
   ```bash
   cp -a /old/path/. /mnt/data/kubernetes/pvc-xxxxx/
   ```
5. Start the service in Kubernetes.
6. Verify, then remove the old compose service.

---

## StatefulSets vs Deployments for databases

For stateless services (Navidrome, Calibre web UI, Paperless web, Firefly web) a `Deployment`
is correct. For stateful services that have a single primary database pod (PostgreSQL, MariaDB,
Redis, Valkey) use a `StatefulSet`. StatefulSets provide:

- Ordered, graceful pod startup and shutdown
- Stable pod names (`pod-0`, `pod-1`, ...)
- Per-pod `volumeClaimTemplates` (the PVC is tied to the pod identity)

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: paperless-db
  namespace: homelab
spec:
  serviceName: paperless-db
  replicas: 1
  selector:
    matchLabels:
      app: paperless-db
  template:
    metadata:
      labels:
        app: paperless-db
    spec:
      containers:
        - name: postgres
          image: postgres:17.5-bookworm
          env:
            - name: POSTGRES_DB
              valueFrom:
                secretKeyRef:
                  name: paperless-db-secret
                  key: POSTGRES_DB
            - name: POSTGRES_USER
              valueFrom:
                secretKeyRef:
                  name: paperless-db-secret
                  key: POSTGRES_USER
            - name: POSTGRES_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: paperless-db-secret
                  key: POSTGRES_PASSWORD
          volumeMounts:
            - name: data
              mountPath: /var/lib/postgresql/data
  volumeClaimTemplates:
    - metadata:
        name: data
      spec:
        accessModes: [ReadWriteOnce]
        storageClassName: local-path
        resources:
          requests:
            storage: 5Gi
```

---

## NFS for shared read-only data (Navidrome music)

Navidrome mounts the music library read-only. If the music lives on a NAS or a shared
directory that multiple pods might eventually need, consider an NFS `PersistentVolume`:

```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: navidrome-music
spec:
  capacity:
    storage: 1Ti
  accessModes:
    - ReadOnlyMany    # multiple pods can mount read-only simultaneously
  nfs:
    server: 192.168.1.50
    path: /share/music
  mountOptions:
    - nfsvers=4.1
```

For now, a simple `local` PV pointing to the existing music directory works fine.
