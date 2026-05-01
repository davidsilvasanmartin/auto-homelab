# Grimmory on Kubernetes — Deploy

This page assumes you've completed [02-setup.md](02-setup.md): Minikube is running, `minikube mount`
is active in a background tab, and your PVCs show `Bound`.

---

## Quick sanity check

Run these before proceeding. Each should give a clean answer.

```sh
# Cluster is up
kubectl get nodes
# NAME       STATUS   ROLES           AGE
# minikube   Ready    control-plane   …

# Mount is active (the directory exists inside Minikube)
minikube ssh "ls /test-data/grimmory"
# app-data  books  mariadb

# PVCs are bound
kubectl get pvc -n grimmory
# All three should show STATUS=Bound
```

---

## Inspecting the chart's values before installing

Before you install anything, it's worth knowing what knobs the chart exposes. Helm gives you a
command for this:

```sh
helm show values /tmp/grimmory-src/deploy/helm/grimmory
```

This prints the full `values.yaml` — every configurable option with its default. Scroll through it
to understand what's available. Our override file (`kubernetes/grimmory/values-override.yaml`)
only changes the settings we need to change for a local test; everything else keeps its default.

---

## The values override file explained

Open `kubernetes/grimmory/values-override.yaml`. Here's what each block does:

```yaml
service:
  type: NodePort      # makes the service reachable from outside the cluster
  nodePort: 30001     # the port you'll visit in your browser: $(minikube ip):30001
```

Without this, the default is `ClusterIP`, which is only reachable from inside the cluster.

```yaml
persistence:
  data:
    existingClaim: grimmory-app-data   # use OUR PVC, not a new auto-generated one
  books:
    existingClaim: grimmory-books
```

By naming `existingClaim`, we tell Helm "don't create new PVCs — bind to these ones we already
made." Those PVCs are bound to our hostPath PVs which point into `./test-data`.

```yaml
mariadb:
  auth:
    database: grimmory
    username: grimmory
    password: "ChangeMe_Grimmory_2025!"
    rootPassword: "ChangeMe_MariaDBRoot_2025!"
  primary:
    persistence:
      existingClaim: grimmory-mariadb   # same idea for the database volume
```

Bitnami's MariaDB sub-chart receives these values as its own configuration. Grimmory's chart
automatically constructs the JDBC URL (`jdbc:mariadb://…`) from the auth values and passes it to
the Grimmory container as an environment variable.

---

## Install with Helm

```sh
helm install grimmory /tmp/grimmory-src/deploy/helm/grimmory \
  --namespace grimmory \
  --values kubernetes/grimmory/values-override.yaml
```

Breaking down the command:

| Part                                        | Meaning                                              |
|---------------------------------------------|------------------------------------------------------|
| `helm install`                              | Create a new release                                 |
| `grimmory`                                  | The release name (`helm list`, `helm upgrade`, etc.) |
| `/tmp/grimmory-src/deploy/helm/grimmory`    | Path to the chart directory                          |
| `--namespace grimmory`                      | Install into the `grimmory` namespace                |
| `--values …`                                | Merge our overrides on top of the chart's defaults   |

Helm will print a summary of what it created:

```
NAME: grimmory
LAST DEPLOYED: …
NAMESPACE: grimmory
STATUS: deployed
REVISION: 1
```

---

## Watch the pods start up

```sh
kubectl get pods -n grimmory --watch
```

You'll see something like this evolve over ~2 minutes:

```
NAME                        READY   STATUS            RESTARTS
grimmory-mariadb-0          0/1     ContainerCreating 0
grimmory-mariadb-0          0/1     Running           0
grimmory-mariadb-0          1/1     Running           0       ← MariaDB is ready
grimmory-…-abc123           0/1     Init:0/1          0       ← waiting for DB
grimmory-…-abc123           0/1     PodInitializing   0
grimmory-…-abc123           0/1     Running           0
grimmory-…-abc123           1/1     Running           0       ← Grimmory is ready
```

Press `Ctrl+C` to stop watching once both pods are `1/1 Running`.

> **Why does Grimmory wait for MariaDB?** The chart includes an init container — a short-lived
> container that runs *before* the main container starts. It loops until MariaDB's port 3306 accepts
> connections, then exits and lets Grimmory start. This prevents Grimmory from crashing on startup
> because the database isn't ready yet.

---

## Open the app

With the Docker driver on macOS, the cluster runs inside a Docker container that is not directly
reachable from your browser. Use `minikube service` to create a localhost tunnel automatically:

```sh
minikube service grimmory -n grimmory
```

Minikube opens your browser at a `http://127.0.0.1:<random-port>` URL that proxies to port 30001
inside the cluster. To get the URL without auto-opening the browser:

```sh
minikube service grimmory -n grimmory --url
# http://127.0.0.1:xxxxx
```

> **Why not `minikube ip`?** On macOS, Docker containers run inside Docker Desktop's own Linux VM.
> The IP returned by `minikube ip` is the container's address inside that VM — not reachable from
> macOS directly. `minikube service` handles the tunnelling for you.

---

## Useful commands while running

```sh
# See all resources in the grimmory namespace
kubectl get all -n grimmory

# Follow Grimmory's logs in real time
kubectl logs -n grimmory -l app.kubernetes.io/name=grimmory -f

# Follow MariaDB's logs
kubectl logs -n grimmory -l app.kubernetes.io/name=mariadb -f

# Open a shell inside the Grimmory pod (useful for debugging)
kubectl exec -it -n grimmory deploy/grimmory -- /bin/sh

# See what Helm actually deployed (rendered YAML, no dry-run)
helm get manifest grimmory -n grimmory
```

---

## Updating the deployment

If you change `values-override.yaml` (e.g. to update the image tag), apply the changes with:

```sh
helm upgrade grimmory /tmp/grimmory-src/deploy/helm/grimmory \
  --namespace grimmory \
  --values kubernetes/grimmory/values-override.yaml
```

Helm will only restart the pods that are affected by the change.

---

## Teardown

```sh
# Remove Grimmory and MariaDB from the cluster (keeps PVs and the test-data directories)
helm uninstall grimmory -n grimmory

# Also remove the namespace and storage objects
kubectl delete namespace grimmory
kubectl delete pv grimmory-app-data grimmory-books grimmory-mariadb
```

The PVs have `Retain` policy so your `./test-data/grimmory/` files are not deleted. To start
completely fresh, clear those directories manually after deleting the PVs.

---

## Troubleshooting

### PVCs stuck in `Pending`

```sh
kubectl describe pvc grimmory-app-data -n grimmory
```

Look at the `Events` section at the bottom. Common causes:

- `storageClassName` mismatch — the PVC has `""` but the PV doesn't. Check `kubectl describe pv grimmory-app-data`.
- The PV was already claimed by a different PVC from a previous attempt. Delete the PV and recreate it.

### MariaDB pod stuck in `CrashLoopBackOff`

```sh
kubectl logs -n grimmory grimmory-mariadb-0
```

If you see permission errors on `/var/lib/mysql`, the `./test-data/grimmory/mariadb/` directory on
your Mac may have files from a failed previous init. Clear it:

```sh
rm -rf test-data/grimmory/mariadb/*
kubectl rollout restart statefulset/grimmory-mariadb -n grimmory
```

### Grimmory pod stuck at `Init:0/1`

The init container is waiting for MariaDB to be healthy. Check MariaDB first:

```sh
kubectl get pods -n grimmory
# if grimmory-mariadb-0 is not 1/1 Ready, debug that first
```

### `helm install` fails with "release already exists"

A previous install attempt left a failed release. Remove it and try again:

```sh
helm uninstall grimmory -n grimmory
helm install grimmory /tmp/grimmory-src/deploy/helm/grimmory \
  --namespace grimmory \
  --values kubernetes/grimmory/values-override.yaml
```

### Grimmory loads but shows a database connection error

The credentials in `values-override.yaml` must match exactly on both sides
(`mariadb.auth.password` and `mariadb.auth.username`). If you changed them after a first install,
MariaDB's data directory already has the old credentials baked in. Clear
`test-data/grimmory/mariadb/` and reinstall to reset.

### `minikube mount` disappeared

If you closed the terminal tab running `minikube mount`, pods that need storage will hang or error.
Re-run the mount command and then restart the affected pods:

```sh
minikube mount "$(pwd)/test-data:/test-data"   # in a new tab
kubectl rollout restart deployment/grimmory -n grimmory
kubectl rollout restart statefulset/grimmory-mariadb -n grimmory
```
