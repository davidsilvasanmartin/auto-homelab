# Grimmory on Kubernetes — Setup

Everything you need to do once on a new machine or a new server before running
`just k8s-up` for the first time.

---

## 1. Install prerequisites

### Dev (macOS)

Docker Desktop must be installed and **running** — Minikube uses it as its container engine.

```sh
brew install minikube kubernetes-cli
```

Grimmory is a JVM app and MariaDB also needs headroom. Open Docker Desktop →
**Settings → Resources → Advanced** and set memory to at least **4 GB** (6 GB is comfortable).

### Prod (Debian)

Install `kubectl` and ensure the cluster is already up (kubeadm init done). The Go `k8s` commands
use the kubeconfig at `~/.kube/config` by default.

```sh
# Install kubectl (Debian)
sudo apt-get install -y kubectl
```

---

## 2. Create your secret

The Secret is **never committed to git**. You create it once per environment.

```sh
# 1. Copy the template
cp kubernetes/grimmory/secret.template.yaml kubernetes/grimmory/secret.yaml

# 2. Edit secret.yaml — replace every CHANGE_ME with real passwords.
#    The namespace must already exist before applying (step 4 below creates it).
```

> **Prod only**: For production, prefer creating the Secret imperatively so the credentials never
> touch the filesystem as a YAML file:
>
> ```sh
> kubectl create secret generic grimmory-db-credentials \
>   --namespace grimmory \
>   --from-literal=db-name=grimmory \
>   --from-literal=db-user=grimmory \
>   --from-literal=db-password='your-password' \
>   --from-literal=db-root-password='your-root-password' \
>   --from-literal=database-url='jdbc:mariadb://mariadb:3306/grimmory'
> ```

---

## 3. Configure prod storage (prod only)

Edit `kubernetes/grimmory/storage/pvs.prod.yaml`. Replace the two placeholders in every PV:

```yaml
nfs:
  server: 192.168.1.X   # ← your NFS server IP
  path: /exports/grimmory/app-data   # ← actual export path on the server
```

You also need to ensure the NFS exports exist on the server before applying:

```sh
# On the NFS server
sudo mkdir -p /exports/grimmory/{app-data,books,bookdrop,mariadb}
sudo exportfs -a
```

Dev users skip this step entirely — `pvs.dev.yaml` uses hostPath, and the `k8s up` command creates
the directories automatically.

---

## 4. Start the cluster

```sh
just k8s-up
```

Once both pods show `1/1 Running`, open the app:

```sh
minikube service grimmory -n grimmory   # dev only — opens a browser tunnel
```

Stop the cluster (data is preserved):

```sh
just k8s-down
```

Stop and delete all persistent data:

```sh
just k8s-wipe
```

---

## What "just k8s-up" does (ordered steps)

The Go `k8s up` command executes the following operations in order. If any step fails,
execution stops and the error is reported.

| # | Step | Dev | Prod |
|---|---|---|---|
| 1 | Start Minikube | `minikube start --driver=docker` | skip |
| 2 | Create test-data directories | `mkdir -p test-data/grimmory/{app-data,books,bookdrop,mariadb}` | skip |
| 3 | Start minikube mount (background) | `minikube mount <project-root>/test-data:/test-data` | skip |
| 4 | Apply namespace | `kubectl apply -f namespace.yaml` | same |
| 5 | Apply ConfigMap | `kubectl apply -f configmap.yaml` | same |
| 6 | Apply Secret | `kubectl apply -f secret.yaml` | same (or already created in step 2) |
| 7 | Apply PVs | `kubectl apply -f storage/pvs.dev.yaml` | `pvs.prod.yaml` |
| 8 | Apply PVCs | `kubectl apply -f storage/pvcs.yaml` | same |
| 9 | Apply MariaDB | `kubectl apply -f mariadb/` | same |
| 10 | Apply Grimmory | `kubectl apply -f grimmory/` | same |
| 11 | Wait for pods ready | Kubernetes API watch | same |

`kubectl apply` is idempotent — re-running `just k8s-up` after changing a YAML file only affects
resources whose spec actually changed. Kubernetes rolls the affected Deployment automatically.

---

## What "just k8s-down" does

| # | Step | Dev | Prod |
|---|---|---|---|
| 1 | Delete namespace (cascades to all resources) | `kubectl delete namespace grimmory` | same |
| 2 | Delete PVs (cluster-scoped, not deleted by namespace removal) | `kubectl delete pv ...` | same |
| 3 | Stop minikube mount | kill background process | skip |
| 4 | Stop Minikube | `minikube stop` | skip |
| 5 | `--wipe`: delete test-data | `rm -rf test-data/grimmory` | skip |

Continue to [03-deploy.md](03-deploy.md) for runtime commands, troubleshooting, and the Go
implementation guide.
