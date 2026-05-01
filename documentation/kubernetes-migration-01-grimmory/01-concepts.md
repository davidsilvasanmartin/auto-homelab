# Grimmory on Kubernetes — Concepts

Before running any commands, this page explains what Kubernetes actually is and why each piece
exists. Skip to [02-setup.md](02-setup.md) if you're already comfortable with Pods, Services, and
PVCs.

---

## What Kubernetes is (in one paragraph)

Kubernetes is a system that runs your containers and keeps them running. You describe *what you
want* (e.g. "I want one Grimmory container always running, listening on port 6060"), and Kubernetes
continuously makes the real world match that description — restarting containers that crash,
rescheduling them if a machine dies, etc. You talk to Kubernetes by submitting YAML files that
describe your desired state.

---

## The building blocks

### Pod

A Pod is the smallest deployable unit in Kubernetes. It wraps one or more containers that need to
run together and share the same network and storage. For Grimmory, each Pod is just one container.

Think of a Pod as a single running process. If it crashes, it's gone — a higher-level object
(Deployment) is responsible for recreating it.

### Deployment

A Deployment says "always keep N copies of this Pod running." If your Pod crashes or you delete it,
the Deployment notices and creates a new one automatically.

```
Deployment (the recipe + the "keep 1 running" rule)
  └── Pod (the actual running container)
```

We have two Deployments: one for Grimmory and one for MariaDB.

### Service

This is the piece that trips up newcomers most. Here's the problem it solves:

Every Pod gets a random IP address when it starts, and a different one when it restarts. So if
Grimmory wants to talk to MariaDB at `10.244.0.5:3306`, that address might be wrong tomorrow after
MariaDB restarts.

A **Service** is a stable, named network endpoint that sits in front of one or more Pods. It always
resolves to the current healthy Pod, no matter how many times it has restarted or what IP it has.

```
Grimmory pod  ──connects to──►  Service "mariadb" (stable)
                                   └──► MariaDB Pod (IP changes, Service doesn't)
```

Inside the cluster, Pods find each other by Service name (e.g. `mariadb:3306`), not by IP.

Services also control how traffic reaches the cluster from outside:

| Service type | Reachable from |
|---|---|
| **ClusterIP** (default) | Only inside the cluster |
| **NodePort** | Outside the cluster via a tunnel or node IP |
| **LoadBalancer** | Outside via a cloud load balancer |

We expose Grimmory as a `NodePort`. MariaDB stays `ClusterIP` — only Grimmory needs to reach it.

### Namespace

A Namespace is a virtual partition inside a cluster. Resources in different namespaces don't
conflict: you can have a Service named `grimmory` in the `grimmory` namespace and another one in
`paperless` without collision.

Everything for Grimmory lives in a namespace called `grimmory`.

### PersistentVolume and PersistentVolumeClaim

Containers are ephemeral — their local filesystem is wiped when the container restarts. Databases
and book libraries need durable storage that outlives the container.

- A **PersistentVolume (PV)** represents a real piece of storage: a directory on disk, an NFS
  share, a cloud disk, etc.
- A **PersistentVolumeClaim (PVC)** is a request for storage: "I need 5 Gi, read-write." Kubernetes
  binds it to a matching PV.
- The Pod mounts the PVC like a normal directory.

```
Pod ──mounts──► PVC "grimmory-app-data" ──bound to──► PV ──backed by──► real storage
```

This is where dev and production diverge:

| Environment | PV backed by |
|---|---|
| Dev (this guide) | A directory on your Mac, exposed to Minikube via `minikube mount` |
| Production | An NFS or SMB share on the Debian server |

The PVCs and everything above them are identical in both environments. Only the PV definitions
change. This is intentional: it keeps the Deployment manifests environment-agnostic.

### Secret

A Secret holds sensitive configuration (passwords, tokens) as key-value pairs. Kubernetes makes
them available to Pods as environment variables or mounted files, without putting them in plain text
in your YAML.

The `secret.yaml` in this repo holds the MariaDB credentials and JDBC URL that Grimmory reads at
startup.

### Init container

An init container runs and completes *before* the main container starts. Grimmory's Deployment
includes one that loops until MariaDB's port 3306 accepts connections, then exits. This prevents
Grimmory from crashing on startup because the database wasn't ready yet.

---

## How the manifests are organised

All Kubernetes YAML for Grimmory lives in `kubernetes/grimmory/`:

```
kubernetes/grimmory/
├── namespace.yaml              # the grimmory namespace
├── secret.yaml                 # DB credentials
├── storage/
│   ├── pvs.yaml                # PersistentVolumes (dev: hostPath)
│   └── pvcs.yaml               # PersistentVolumeClaims (same in dev and prod)
├── mariadb/
│   ├── deployment.yaml
│   └── service.yaml
└── grimmory/
    ├── deployment.yaml
    └── service.yaml
```

These are plain Kubernetes manifests — no templating engine, no package manager. What you read is
exactly what gets applied to the cluster.

> **What about Helm?** Helm is a package manager for Kubernetes that adds templating and versioning
> on top of plain manifests. It makes sense when you're distributing software to others or managing
> many similar deployments. For a single homelab app with custom configuration, plain manifests are
> simpler and more transparent — you always know exactly what's running.

---

## Architecture

```
Your Mac
├── test-data/grimmory/         ← files visible here
│   ├── app-data/
│   ├── books/
│   ├── bookdrop/
│   └── mariadb/
│
└── minikube mount (9p bridge, started by dev-up.sh)
        │
        ▼
Minikube container (Docker)  /test-data/grimmory/
        │
        ▼  hostPath PVs (dev only)
┌──────────────────────────────────────────────┐
│  namespace: grimmory                         │
│                                              │
│  Secret ── grimmory-db-credentials           │
│                                              │
│  Deployment/mariadb ── Service/mariadb       │
│    Pod: mariadb:11.4      (ClusterIP:3306)   │
│    PVC: grimmory-mariadb-data                │
│                                              │
│  Deployment/grimmory ── Service/grimmory     │
│    Pod: grimmory          (NodePort:30001)   │
│    PVC: grimmory-app-data                    │
│    PVC: grimmory-books                       │
│    PVC: grimmory-bookdrop                    │
└──────────────────────────────────────────────┘
        │
        ▼ minikube service tunnel
http://127.0.0.1:<port>  ← open in browser
```

Continue to [02-setup.md](02-setup.md).
