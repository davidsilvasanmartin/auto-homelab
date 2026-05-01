# Grimmory on Kubernetes — Setup

Install all the tools, start Minikube, and prepare storage. This only needs to be done once.

---

## 1. Install dependencies

Docker Desktop must be installed and **running** before you start Minikube — it is the VM engine
Minikube will use.

The remaining tools install via Homebrew:

```sh
# Minikube — the local Kubernetes cluster
brew install minikube

# kubectl — the CLI for talking to Kubernetes
brew install kubernetes-cli

# Helm — the Kubernetes package manager
brew install helm
```

Verify the installs:

```sh
minikube version    # Minikube v1.x.x
kubectl version --client --short   # Client Version: v1.x.x
helm version --short   # v3.x.x
```

---

## 2. Start the Minikube cluster

```sh
minikube start --driver=docker
```

Minikube will use Docker Desktop as its engine and allocate the memory and CPUs you have already
configured in Docker Desktop's settings (Resources → Advanced). If you want to override them for
this cluster:

```sh
minikube start --driver=docker --memory=4096 --cpus=2
```

> **Why Docker and not QEMU?** QEMU's default network mode (SLIRP) only allows outbound connections
> from the VM, which breaks `minikube mount`. The Docker driver uses Docker's network stack, which
> supports the bidirectional connection that `minikube mount` requires — no extra setup needed.

The first run pulls the Minikube base image (~500 MB) and can take a few minutes. You should see:

```
✅  Done! kubectl is now configured to use "minikube" cluster and "default" namespace by default
```

Verify the cluster is up:

```sh
kubectl get nodes
# NAME       STATUS   ROLES           AGE   VERSION
# minikube   Ready    control-plane   1m    v1.x.x
```

One node called `minikube` in `Ready` state means everything is working.

---

## 3. Mount `./test-data` into Minikube

`minikube mount` creates a live bridge between a directory on your Mac and a path inside the
Minikube container. Any file you write to `./test-data` on your Mac immediately appears at
`/test-data` inside the container — and vice versa.

**Open a new terminal tab** and run (from the project root):

```sh
minikube mount "$(pwd)/test-data:/test-data"
```

Expected output:

```
📁  Mounting host path /Users/dev/Developer/auto-homelab/test-data into VM as /test-data ...
    ▪ Mount type:   9p
    ▪ User ID:      docker
    ▪ Group ID:     docker
    ▪ Version:      9p2000.L
    ▪ Message Size: 262144
    ▪ Bind Address: 127.0.0.1:xxxxx
🚀  Userspace file server: ufs starting
✅  Successfully mounted /Users/.../test-data to /test-data

📌  NOTE: This process must stay alive for the mount to be accessible ...
```

**Keep this terminal tab open.** The mount process must stay running for pods to access the files.
If you stop it, pods that need storage will hang or crash.

> **Tip:** Bookmark this terminal. Any time you restart your Mac or close the tab you must re-run
> the mount command before starting Grimmory.

Verify the mount from inside Minikube:

```sh
minikube ssh "ls /test-data"
# grimmory
```

---

## 4. Create the test-data directories

The directories need to exist before the pods start. Run once from the project root:

```sh
mkdir -p test-data/grimmory/app-data \
         test-data/grimmory/books \
         test-data/grimmory/mariadb
```

---

## 5. Pre-create storage (PVs and PVCs)

Helm will create PVCs for us, but we want to control exactly where the data lands (inside
`./test-data`). The way to do that is to create the PVs and PVCs *before* Helm runs, and then tell
Helm to use the ones we already made (via `existingClaim`).

Apply the storage manifests (these are cluster-scoped PVs, so no namespace needed for that file):

```sh
kubectl apply -f kubernetes/grimmory/storage/pvs.yaml
kubectl apply -f kubernetes/grimmory/namespace.yaml
kubectl apply -f kubernetes/grimmory/storage/pvcs.yaml
```

Check everything bound correctly — all PVCs should show `STATUS=Bound`:

```sh
kubectl get pvc -n grimmory
```

Expected output:

```
NAME                   STATUS   VOLUME                  CAPACITY   ACCESS MODES
grimmory-app-data      Bound    grimmory-app-data        5Gi        RWO
grimmory-books         Bound    grimmory-books           50Gi       RWO
grimmory-mariadb       Bound    grimmory-mariadb         5Gi        RWO
```

If a PVC shows `Pending` instead of `Bound`, see the troubleshooting section in
[03-deploy.md](03-deploy.md).

---

## 6. Get the Grimmory Helm chart

The chart lives inside the Grimmory source repository. Clone it somewhere outside the project
(we don't want to commit it here):

```sh
git clone https://github.com/grimmory-tools/grimmory.git /tmp/grimmory-src
```

Download the chart's dependencies (Bitnami's MariaDB sub-chart):

```sh
helm dependency update /tmp/grimmory-src/deploy/helm/grimmory
```

You should see a `charts/` directory appear inside the chart with a MariaDB `.tgz` file:

```sh
ls /tmp/grimmory-src/deploy/helm/grimmory/charts/
# mariadb-22.x.x.tgz
```

---

## What `helm dependency update` actually did

Grimmory's `Chart.yaml` declares:

```yaml
dependencies:
  - name: mariadb
    version: 22.0.*
    repository: oci://registry-1.docker.io/bitnamicharts
```

`helm dependency update` read that, pulled Bitnami's MariaDB chart from Docker Hub's OCI registry,
and placed it in `charts/`. When you run `helm install` next, Helm installs both the Grimmory
templates *and* the MariaDB sub-chart in one shot — MariaDB does not need to be set up separately.

Continue to [03-deploy.md](03-deploy.md).
