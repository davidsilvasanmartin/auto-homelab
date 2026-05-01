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

In this guide we create two Deployments: one for Grimmory and one for MariaDB.

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
| **NodePort** | Outside the cluster via `<node-ip>:<port>` |
| **LoadBalancer** | Outside via a cloud load balancer |

For this test we expose Grimmory as a `NodePort` on port 30001. MariaDB stays `ClusterIP` — only
Grimmory needs to reach it.

### Namespace

A Namespace is a virtual partition inside a cluster. Resources in different namespaces don't
conflict: you can have a Service named `grimmory` in the `grimmory` namespace and another one in
`paperless` without collision.

We put everything for Grimmory in a namespace called `grimmory`.

### PersistentVolume and PersistentVolumeClaim

Containers are ephemeral — their local filesystem is wiped when the container restarts. Databases
and book libraries need durable storage that outlives the container.

- A **PersistentVolume (PV)** represents a real piece of storage: a directory on disk, an NFS
  share, a cloud disk, etc.
- A **PersistentVolumeClaim (PVC)** is a request for storage: "I need 5 Gi, read-write." Kubernetes
  binds it to a matching PV.
- The Pod mounts the PVC like a normal directory.

```
Pod ──mounts──► PVC "grimmory-app-data" ──bound to──► PV ──backed by──► /test-data/grimmory/app-data
```

In this guide we pre-create `hostPath` PVs that point into `./test-data/` on your Mac (exposed to
Minikube via `minikube mount`).

### Secret

A Secret holds sensitive configuration (passwords, tokens) as base64-encoded key-value pairs.
Kubernetes makes them available to Pods as environment variables or mounted files, without putting
them in plain text in your YAML.

In this guide a Secret holds the MariaDB username, password, and JDBC URL that Grimmory reads on
startup.

---

## What Helm is

Helm is the package manager for Kubernetes — think Homebrew, but for k8s applications.

Without Helm, deploying Grimmory means writing and applying seven or more individual YAML files:
Namespace, Secret, two PVs, two PVCs, two Deployments, two Services. Helm bundles all of that into
a single **chart** and lets you install it with one command.

### Charts

A chart is a directory of templates. Each template is a YAML file with placeholders:

```yaml
# templates/service.yaml (simplified)
apiVersion: v1
kind: Service
metadata:
  name: {{ .Release.Name }}-grimmory
spec:
  type: {{ .Values.service.type }}
  ports:
    - port: {{ .Values.service.port }}
```

### Values

`values.yaml` inside the chart provides defaults for all those placeholders. You override the ones
you care about in your own file (called a *values override*) and pass it to Helm on install:

```sh
helm install grimmory ./chart --values my-overrides.yaml
```

Helm merges your overrides on top of the defaults and renders the final YAML.

### Releases

When you run `helm install`, Helm creates a **release** — a named, versioned installation of a
chart. You can upgrade it (`helm upgrade`), roll it back (`helm rollback`), or delete it
(`helm uninstall`).

```
Chart (the recipe)  +  Values (your config)  =  Release (running in the cluster)
```

### Dependencies

A chart can declare that it depends on other charts. Grimmory's chart depends on Bitnami's MariaDB
chart. Running `helm dependency update` downloads those sub-charts so you can install everything
in one go — no need to set up MariaDB separately.

---

## Architecture for this guide

```
Your Mac
├── ./test-data/grimmory/   ← files visible here
│   ├── app-data/
│   ├── books/
│   └── mariadb/
│
└── minikube mount (live bridge)
        │
        ▼
Minikube container (Docker)  /test-data/grimmory/
        │
        ▼  hostPath PVs
┌──────────────────────────────────────────────┐
│  namespace: grimmory                         │
│                                              │
│  Secret ── grimmory-db-credentials           │
│                                              │
│  Deployment/mariadb ── Service/mariadb       │
│    Pod: mariadb:11        (ClusterIP:3306)   │
│    PVC: grimmory-mariadb                     │
│                                              │
│  Deployment/grimmory ── Service/grimmory     │
│    Pod: grimmory/grimmory  (NodePort:30001)  │
│    PVC: grimmory-app-data                    │
│    PVC: grimmory-books                       │
└──────────────────────────────────────────────┘
        │
        ▼ NodePort / minikube service tunnel
http://127.0.0.1:<port>  ← you open this in your browser
```

Continue to [02-setup.md](02-setup.md).
