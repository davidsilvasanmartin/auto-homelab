# Grimmory on Kubernetes — Concepts & Architecture

This page explains the architecture decisions behind the Kubernetes setup: what runs where, how
storage works across environments, and how configuration is managed. Skip to
[02-setup.md](02-setup.md) if you already know Kubernetes well.

---

## What Kubernetes is (in one paragraph)

Kubernetes is a system that runs your containers and keeps them running. You describe *what you
want* (e.g. "I want one Grimmory container always running, listening on port 6060"), and Kubernetes
continuously reconciles reality against that description — restarting crashed containers,
rescheduling them if a node dies. You communicate with Kubernetes by applying YAML manifests that
describe desired state.

---

## The building blocks

### Pod

The smallest deployable unit. It wraps one or more containers that share the same network and
storage. For Grimmory, every Pod is a single container.

Pods are ephemeral. If a Pod crashes it is gone — a Deployment is responsible for recreating it.

### Deployment

A Deployment declares "always keep N copies of this Pod running." It watches Pod health and creates
new Pods to replace any that disappear.

This project has two Deployments: one for Grimmory, one for MariaDB.

### Service

Every Pod gets a random IP address when it starts, and a different one if it restarts. A Service is
a stable, named endpoint that always routes to the current healthy Pod. Inside the cluster, Pods
find each other by Service name — not by IP.

```
Grimmory pod ──► Service "mariadb" (stable DNS name)
                    └──► MariaDB Pod (IP changes; Service doesn't)
```

MariaDB is `ClusterIP` (reachable only inside the cluster). Grimmory is `NodePort` (reachable from
outside via `minikube service`).

### Namespace

A virtual partition inside a cluster. All Grimmory resources live in the `grimmory` namespace.

### ConfigMap

A key-value store for non-sensitive configuration. Values are injected into Pods as environment
variables. Changing a ConfigMap and rolling the Deployment is the correct way to change application
settings.

This project uses one ConfigMap (`grimmory-config`) for things like timezone and feature flags.

### Secret

Like a ConfigMap but for sensitive values: passwords, connection strings. Kubernetes base64-encodes
the values (this is NOT encryption — it is encoding). Secrets are injected the same way as
ConfigMaps.

**Rule of thumb: if it would be embarrassing to commit it to git, it goes in a Secret, not a
ConfigMap.**

### PersistentVolume and PersistentVolumeClaim

Containers are ephemeral — their local filesystem is wiped on restart. Databases and book libraries
need storage that outlives the container.

- A **PersistentVolume (PV)** represents real storage: a directory, an NFS share, a cloud disk.
- A **PersistentVolumeClaim (PVC)** is a request for storage: "I need 5 Gi, read-write."
  Kubernetes binds the claim to a matching PV.
- The Pod mounts the PVC like a normal directory.

```
Pod ──mounts──► PVC "grimmory-app-data" ──bound to──► PV ──backed by──► real storage
```

### Init container

An init container runs and completes *before* the main container starts. Grimmory's Deployment
includes one that loops until MariaDB's port 3306 accepts connections. This prevents Grimmory from
crashing at startup because the database wasn't ready yet.

---

## The two core design decisions

### 1. Storage: separating environment-specific PVs from environment-agnostic PVCs

The only thing that differs between dev and prod is WHERE storage lives. Everything above the PV
layer (PVCs, Deployments, Services) is identical.

| Layer | Dev (Minikube on macOS) | Prod (Debian + kubeadm) |
|---|---|---|
| PV backend | Host directory inside Minikube VM | NFS share on the home server or NAS |
| PV file | `storage/pvs.dev.yaml` | `storage/pvs.prod.yaml` |
| PVC file | `storage/pvcs.yaml` | `storage/pvcs.yaml` (same file) |
| Deployments | identical | identical |

The Go `k8s up` command applies the correct PV file based on the `--env` flag. Everything else is
applied identically in both environments.

This is the standard pattern: keep environment-specific concerns at the infrastructure layer (PVs),
and keep the application layer (PVCs, Deployments, ConfigMaps) fully portable.

#### Dev storage: minikube mount

Minikube runs inside a Docker container. To make files on your Mac visible inside that container,
`minikube mount` starts a 9P file server that bridges `<project-root>/test-data` into the VM at
`/test-data`. The `hostPath` PVs in `pvs.dev.yaml` then point to `/test-data/grimmory/*` inside the
VM.

The `k8s up` command starts the mount in the background before applying manifests. The mount
process must stay running for pods to access storage — if you reboot your Mac, run `just k8s-up`
again.

#### Prod storage: NFS

Edit `storage/pvs.prod.yaml` and replace the placeholder `nfs.server` IP and `nfs.path` values
with your actual network shares. You only need to do this once per environment.

### 2. Configuration: ConfigMap for settings, Secret for credentials

Non-sensitive settings (timezone, feature flags, disk type) live in `configmap.yaml` and are
applied the same way in both environments. Change a setting by editing the ConfigMap and running
`k8s up` again — the Deployment rolls automatically.

Sensitive values (database passwords, connection strings) live in a Secret. The Secret is **not
committed to git**. The workflow is:

1. Copy `secret.template.yaml` to `secret.yaml` (git-ignored).
2. Fill in real passwords.
3. Apply once: `kubectl apply -f kubernetes/grimmory/secret.yaml`.

The Go `k8s configure` command (to be implemented) will automate steps 1–3 interactively, similar
to how `configure` works for Docker Compose. See [03-deploy.md](03-deploy.md) for the
implementation guide.

This mirrors the standard Kubernetes practice: infrastructure-as-code for everything except
credentials, which are applied imperatively and never stored in source control.

---

## File layout

```
kubernetes/grimmory/
├── namespace.yaml              # the grimmory namespace
├── configmap.yaml              # non-sensitive settings (applied in all environments)
├── secret.template.yaml        # template to copy → secret.yaml (git-ignored)
├── storage/
│   ├── pvs.dev.yaml            # PVs for Minikube (hostPath)
│   ├── pvs.prod.yaml           # PVs for prod (NFS) — edit server IP and paths
│   └── pvcs.yaml               # PVCs — identical in both environments
├── mariadb/
│   ├── deployment.yaml
│   └── service.yaml
└── grimmory/
    ├── deployment.yaml
    └── service.yaml
```

Plain manifests — no templating engine, no package manager. What you read is exactly what gets
applied to the cluster.

> **What about Helm or Kustomize?** Helm is a package manager useful for distributing software to
> others. Kustomize adds overlay-based templating and is built into `kubectl`. Both are valuable
> when environments diverge significantly. For this project, the only difference between dev and
> prod is two YAML files (`pvs.dev.yaml` vs `pvs.prod.yaml`), so plain manifests with explicit
> environment selection in the Go command are simpler and more transparent.

---

## Architecture diagram

```
Your Mac
├── test-data/grimmory/          ← your data (git-ignored)
│   ├── app-data/
│   ├── books/
│   ├── bookdrop/
│   └── mariadb/
│
└── minikube mount (9P bridge, started by "just k8s-up")
        │
        ▼
Minikube VM (Docker container)  /test-data/grimmory/
        │
        ▼  hostPath PVs (pvs.dev.yaml)
┌──────────────────────────────────────────────────────┐
│  namespace: grimmory                                 │
│                                                      │
│  ConfigMap: grimmory-config                          │
│  Secret:    grimmory-db-credentials                  │
│                                                      │
│  Deployment/mariadb ── Service/mariadb               │
│    Pod: mariadb:11.4       ClusterIP:3306            │
│    PVC: grimmory-mariadb-data                        │
│                                                      │
│  Deployment/grimmory ── Service/grimmory             │
│    Pod: grimmory           NodePort:30001            │
│    PVC: grimmory-app-data                            │
│    PVC: grimmory-books                               │
│    PVC: grimmory-bookdrop                            │
└──────────────────────────────────────────────────────┘
        │
        ▼ minikube service tunnel (opened on demand)
http://127.0.0.1:<port>  ← open in browser
```

Continue to [02-setup.md](02-setup.md).
