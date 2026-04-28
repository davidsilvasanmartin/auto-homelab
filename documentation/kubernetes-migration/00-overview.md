# Kubernetes Migration: Overview

This document analyses what a migration from the current Docker Compose homelab to Kubernetes
would look like, what it would improve, what it would complicate, and whether anything is
fundamentally impossible. All subsequent documents in this directory go deeper into specific
areas. The assumed target is a **single Debian node managed by kubeadm**, with Go kept as the
primary language for CLI tooling.

---

## What changes at a high level

| Concept today | Kubernetes equivalent |
|---|---|
| `docker-compose.yml` service block | `Deployment` or `StatefulSet` + `Service` |
| Docker bridge network + static IPs | `ClusterIP` / `NodePort` / `HostNetwork` |
| Named volumes / host-bind mounts | `PersistentVolume` + `PersistentVolumeClaim` |
| `.env` file | `ConfigMap` + `Secret` |
| Traefik label-based routing | Traefik `IngressRoute` CRDs or standard `Ingress` |
| `docker compose up -d` | `kubectl apply -f` (or `helm install`) |
| `--docker-context` flag | `KUBECONFIG` / `kubectl config use-context` |
| Go `docker compose` shell-out | Go Kubernetes client (`k8s.io/client-go`) |
| Manual backup with `docker exec` + `cp` | CronJob pods with shared PVC access |
| `restic` called from host | `restic` run inside a CronJob pod |

---

## What would be massively improved

### 1. The docker-context problem goes away

The biggest day-to-day pain point — managing `--docker-context` flags across all commands — is
replaced by Kubernetes contexts, which are a first-class concept in kubeconfig. Every tool in the
ecosystem (`kubectl`, `helm`, `k9s`, the Go `client-go` library) reads the same kubeconfig file.
You switch contexts once; everything follows.

### 2. The docker-compose.yml sprawl is solved structurally

Right now all services are crammed into one file. Kubernetes manifests are individual files (or
Helm charts) per service, kept in a directory tree. You can apply, delete, or roll back a single
service without touching anything else. This also makes code review much easier.

### 3. Rolling updates and health checks become native

Currently stopping and restarting a service is an all-or-nothing operation. Kubernetes
`Deployments` perform rolling updates by default: it brings up the new pod before removing the
old one, and rolls back automatically if the readiness probe never passes. Every service gets
`livenessProbe` and `readinessProbe` for free.

### 4. Resource limits and quotas

Docker Compose has no enforcement of CPU/memory limits at the compose level. Kubernetes
`resources.requests` and `resources.limits` let you guarantee that one rogue service (e.g.
Immich machine-learning during a large import) cannot starve others.

### 5. ConfigMaps and Secrets replace the `.env` file

The `.env` file is a single flat file on disk. It is passed to every container, even containers
that only need two of the sixty variables. Kubernetes `ConfigMap` and `Secret` objects are
namespace-scoped, can be mounted as files or injected as environment variables per-container,
and can be updated without restarting pods (with some caveats). Secrets are stored separately
from config and can later be backed by an external store (Vault, Sealed Secrets, etc.).

### 6. Go tooling becomes cleaner

The current Go CLI shell-outs to `docker compose` and `docker exec`, which means it has an
implicit dependency on the Docker CLI being present and the docker context being configured
correctly. With `client-go`, the Go binary talks directly to the Kubernetes API server. No
shell dependency, structured errors, and the full Kubernetes object model is available as typed
Go structs.

### 7. Backup CronJobs replace manual invocation

`backup local` currently must be run manually and requires all containers to be up. A
Kubernetes `CronJob` runs on a schedule, mounts the same `PersistentVolumeClaim` as the
service it backs up, runs `pg_dump` or `restic backup` inside the pod, and exits. No host
involvement needed.

### 8. Future-proofing: second node is easy

Adding a second machine to the cluster means more scheduling capacity with no changes to the
service definitions. Docker Compose does not scale beyond one host without Swarm or a similar
layer.

---

## What would be harder or more complex

### 1. Port 53 for AdGuard

This is the most concrete technical challenge of the whole migration. AdGuard needs to bind to
port 53 on the host's network interface so that your router or clients can point DNS at the
server. In Kubernetes, `NodePort` services only expose ports in the `30000–32767` range. There
are three practical solutions, each with trade-offs:

- **`hostNetwork: true`** on the AdGuard pod — the pod shares the node's network namespace,
  port 53 is available directly. Simple, but the pod can now touch any port on the host.
- **`hostPort: 53`** on the container port — similar effect, slightly narrower scope. Less
  recommended by the community.
- **MetalLB** (a load-balancer for bare-metal clusters) — assigns a real IP from a pool to a
  `LoadBalancer` service. AdGuard gets its own IP and port 53. Cleanest, but adds another
  component to maintain.

### 2. Single-node etcd is a SPOF

`kubeadm` by default sets up a single etcd instance. If that node goes down, the cluster
control plane is unavailable. For a homelab this is almost certainly acceptable, but it is a
regression from Docker Compose where "the cluster" is just a running daemon — nothing special
to lose.

### 3. Persistent storage is more complicated

Docker bind mounts (`/path/on/host:/path/in/container`) are trivially simple. The Kubernetes
equivalent on a single node is a `PersistentVolume` of type `local` (or a storage class like
`local-path-provisioner`). It works, but you need to declare `PersistentVolumeClaim` objects,
understand `StorageClass`, and know that a `local` PV is pinned to the node it lives on — which
matters if you ever add a second node.

### 4. Initialisation logic becomes more explicit

AdGuard has a custom Dockerfile and a startup shell script that runs `htpasswd` to set a
password. In Kubernetes this translates to an `initContainer`. Paperless needs its database
before it can start — Kubernetes `initContainers` handle this too, but the logic must be made
explicit as pod spec rather than `depends_on` in compose.

### 5. Overhead for a homelab scale

Kubernetes runs: API server, scheduler, controller manager, etcd, kubelet, kube-proxy, CoreDNS,
and whatever CNI plugin you chose. On a machine with 4–8 GB RAM this is noticeable. Docker
Compose has virtually zero overhead beyond the containers themselves.

### 6. The Go CLI needs a significant rewrite

`docker compose` shell-outs must be replaced with `client-go` API calls. The backup command
must be replaced (or complemented) by CronJob manifests. The configure command would write to
`ConfigMap`/`Secret` objects instead of a `.env` file. This is substantial work, probably
2–4× more code than today, but it becomes dramatically more robust.

---

## Is anything impossible in Kubernetes?

Nothing this project does today is *impossible* in Kubernetes. However, a few things are
significantly harder or require a different approach:

- **Port 53 binding**: Not impossible, but requires one of the three approaches above.
- **`docker exec` for backups**: `kubectl exec` into pods works identically. The Go code
  switches from `docker exec` to a Kubernetes `Exec` subresource call.
- **The AdGuard custom Dockerfile / htpasswd startup**: Init containers replace this cleanly.
- **The `HOMELAB_GENERAL_UID`/`GID` trick in `compose.go`**: In Kubernetes, the pod's
  `securityContext.runAsUser` / `fsGroup` fields handle this at the pod spec level, and
  `local` PVs respect those permissions.

The one area that deserves honest attention is **stateful workloads**: Calibre, Paperless,
Immich, Firefly, and Navidrome all have data on disk. Kubernetes does not make stateful
workloads harder to run — it makes them explicit. `StatefulSets` provide ordered startup,
stable network identities, and per-pod PVCs. They are more verbose than compose but more
correct.

---

## Recommended migration path

Do not attempt a big-bang migration. The recommended path is:

1. Set up the cluster and storage class (see `01-cluster-setup.md`)
2. Migrate Traefik first — it is stateless and gives you ingress for everything else
3. Migrate cert-manager to handle TLS (replaces Traefik's `certresolver`)
4. Migrate AdGuard, choosing your port-53 strategy
5. Migrate stateless services one by one (Navidrome, Calibre, Portainer)
6. Migrate stateful services (Paperless, Firefly, Immich) last
7. Migrate the Go CLI to use `client-go`
8. Replace manual backup commands with CronJobs

Running the Docker Compose stack in parallel on the same machine during migration is possible
if you assign the K8s ingress to a different IP or port temporarily.

---

## Document index

| File | Topic |
|---|---|
| `00-overview.md` | This file — trade-offs and migration strategy |
| `01-cluster-setup.md` | kubeadm on Debian, CNI, storage class |
| `02-networking.md` | Services, Ingress, AdGuard port 53 |
| `03-storage.md` | PersistentVolumes, local-path-provisioner |
| `04-services.md` | Per-service manifest breakdown |
| `05-tls.md` | cert-manager replacing Traefik certresolver |
| `06-go-tooling.md` | Rewriting the CLI with client-go |
| `07-backup.md` | CronJob-based backup replacing backup local/cloud |
