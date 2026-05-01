# Grimmory on Kubernetes — Runtime & Troubleshooting

This page assumes `scripts/dev-up.sh` has completed and both pods are `1/1 Running`.

---

## Opening the app

```sh
minikube service grimmory -n grimmory
```

Minikube creates a localhost tunnel and opens your browser. To get the URL without auto-opening:

```sh
minikube service grimmory -n grimmory --url
# http://127.0.0.1:xxxxx
```

> On macOS with the Docker driver, `minikube ip` returns an address inside Docker Desktop's
> internal VM — not reachable from your browser. Always use `minikube service` instead.

---

## Useful commands

```sh
# All resources in the grimmory namespace
kubectl get all -n grimmory

# Stream Grimmory's logs
kubectl logs -n grimmory -l app=grimmory -f

# Stream MariaDB's logs
kubectl logs -n grimmory -l app=mariadb -f

# Shell inside the Grimmory pod
kubectl exec -it -n grimmory deploy/grimmory -- /bin/sh

# Check PVC binding
kubectl get pvc -n grimmory

# Check PVs (cluster-scoped, no -n needed)
kubectl get pv
```

---

## Reapplying after a manifest change

`kubectl apply` is idempotent — just re-run the startup script:

```sh
scripts/dev-up.sh
```

Kubernetes will only restart pods whose spec actually changed.

---

## Teardown

```sh
# Stop everything, keep test-data/
scripts/dev-down.sh

# Stop everything and delete all data
scripts/dev-down.sh --wipe
```

`--wipe` deletes the MariaDB data directory, which resets the database on the next `dev-up.sh`.
Without it, your data persists across restarts.

---

## Troubleshooting

### A PVC shows `Pending`

```sh
kubectl describe pvc <name> -n grimmory
```

Look at the `Events` section. Common causes:

- **`storageClassName` mismatch** — the PVC has `storageClassName: ""` (static binding) but the PV
  was created with a non-empty class. Check `kubectl describe pv <name>`.
- **Name mismatch** — the `volumeName` in the PVC doesn't match the PV's metadata name. Compare
  `storage/pvs.yaml` and `storage/pvcs.yaml`.
- **PV already claimed** — a PV can only bind to one PVC. If it shows `Released` from a previous
  attempt, delete it and reapply: `kubectl delete pv <name> && kubectl apply -f kubernetes/grimmory/storage/pvs.yaml`.

### MariaDB stuck in `CrashLoopBackOff`

```sh
kubectl logs -n grimmory -l app=mariadb
```

If you see permission errors on `/var/lib/mysql`, the MariaDB data directory has leftover files
from a failed first initialisation. Clear it and restart:

```sh
rm -rf test-data/grimmory/mariadb/*
kubectl rollout restart deployment/mariadb -n grimmory
```

### Grimmory stuck at `Init:0/1`

The init container is waiting for MariaDB to accept connections. Check MariaDB first:

```sh
kubectl get pods -n grimmory
# grimmory-mariadb-xxx should be 1/1 Ready before Grimmory starts
```

### `minikube mount` stopped (pods hang or error)

```sh
# Check if the mount process is still running
cat /tmp/minikube-mount-grimmory.pid | xargs kill -0 2>/dev/null && echo running || echo stopped
```

If stopped, re-run the startup script — it detects the stale PID and restarts the mount:

```sh
scripts/dev-up.sh
```

### Grimmory loads but shows a database connection error

The credentials in `secret.yaml` must match what MariaDB initialised with. If you changed them
after a first run, MariaDB's data directory still has the old credentials baked in. Wipe and
restart:

```sh
scripts/dev-down.sh --wipe
scripts/dev-up.sh
```
