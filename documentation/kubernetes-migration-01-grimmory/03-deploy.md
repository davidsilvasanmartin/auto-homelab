# Grimmory on Kubernetes — Runtime, Troubleshooting & Go Implementation

---

## Day-to-day commands

```sh
just k8s-up                                   # start the cluster (idempotent)
just k8s-down                                 # stop, keep data
just k8s-wipe                                 # stop, delete all data

minikube service grimmory -n grimmory         # open browser tunnel (dev only)
minikube service grimmory -n grimmory --url   # get URL without opening browser

kubectl get all -n grimmory                   # overview of all resources
kubectl get pods -n grimmory -w               # watch pod status

kubectl logs -n grimmory -l app=grimmory -f   # stream Grimmory logs
kubectl logs -n grimmory -l app=mariadb -f    # stream MariaDB logs

kubectl exec -it -n grimmory deploy/grimmory -- /bin/sh   # shell into Grimmory pod

kubectl get pvc -n grimmory                   # check PVC binding
kubectl get pv                                # check PVs (cluster-scoped)
```

---

## Troubleshooting

### A PVC shows `Pending`

```sh
kubectl describe pvc <name> -n grimmory
```

Check the `Events` section. Common causes:

- **`storageClassName` mismatch** — the PVC requests `storageClassName: ""` (static binding) but
  the PV was created with a non-empty class. Check `kubectl describe pv <name>`.
- **Name mismatch** — the `volumeName` in the PVC doesn't match the PV's `metadata.name`. Compare
  `storage/pvcs.yaml` with `pvs.dev.yaml` or `pvs.prod.yaml`.
- **PV already claimed** — a PV can only bind to one PVC. If it shows `Released`, delete and
  reapply:
  ```sh
  kubectl delete pv <name>
  kubectl apply -f kubernetes/grimmory/storage/pvs.dev.yaml
  ```

### MariaDB stuck in `CrashLoopBackOff`

```sh
kubectl logs -n grimmory -l app=mariadb
```

If you see permission errors on `/var/lib/mysql`, the data directory has leftover files from a
failed first initialisation. Wipe and restart:

```sh
rm -rf test-data/grimmory/mariadb/*
kubectl rollout restart deployment/mariadb -n grimmory
```

### Grimmory stuck at `Init:0/1`

The init container is waiting for MariaDB. Check MariaDB first:

```sh
kubectl get pods -n grimmory   # mariadb pod must be 1/1 Ready before Grimmory starts
```

### `minikube mount` stopped (pods hang)

```sh
kill -0 "$(cat /tmp/minikube-mount-grimmory.pid)" 2>/dev/null && echo running || echo stopped
```

If stopped, `just k8s-up` detects the stale PID and restarts the mount.

### Credentials mismatch after first run

If you changed the Secret after MariaDB already initialized, the data directory still has the old
credentials baked in. Wipe and restart:

```sh
just k8s-wipe
just k8s-up
```

---

## Go implementation guide

This section is for contributors implementing the `k8s` Cobra subcommand. The goal is a
cross-platform, testable alternative to the shell scripts this project is moving away from.

### Package layout

```
cmd/
└── k8s.go           # Cobra: "go run . k8s up|down [--env dev|prod] [--wipe]"

internal/
└── kubernetes/
    ├── client.go    # build a client-go dynamic client from kubeconfig
    ├── ops.go       # Op interface + concrete types (ApplyOp, DeleteOp, EnsureDirOp, ...)
    └── runner.go    # RunOps: execute a slice of Ops in order, stop on first error
```

### The Op interface and command queue

The core pattern is an ordered slice of operations executed sequentially. This replaces shell
scripts with type-safe, individually testable units.

```go
// internal/kubernetes/ops.go

type Op interface {
    Name() string
    Run(ctx context.Context) error
}
```

Example operations:

```go
// Apply a manifest file via the Kubernetes API
type ApplyOp struct {
    Path string // relative to project root, e.g. "kubernetes/grimmory/namespace.yaml"
}

// Delete a Kubernetes resource by kind and name
type DeleteOp struct {
    Kind      string
    Name      string
    Namespace string // empty for cluster-scoped resources (PVs)
}

// Create a local directory if it does not exist
type EnsureDirOp struct {
    Path string
}

// Run an external command (for minikube, which has no Go API)
type ExecOp struct {
    Cmd        string
    Args       []string
    IgnoreErr  bool // set true for idempotent checks (e.g. "minikube status")
}

// Start a long-running background process and track its PID
type BackgroundExecOp struct {
    Cmd     string
    Args    []string
    PIDFile string
}

// Wait for all pods in a namespace to be Ready
type WaitPodsReadyOp struct {
    Namespace string
    Timeout   time.Duration
}
```

The runner is trivial:

```go
// internal/kubernetes/runner.go

func RunOps(ctx context.Context, ops []Op) error {
    for _, op := range ops {
        slog.Info("running operation", "name", op.Name())
        if err := op.Run(ctx); err != nil {
            return fmt.Errorf("operation %q failed: %w", op.Name(), err)
        }
    }
    return nil
}
```

### Kubernetes client (client-go)

Use the Kubernetes Go client library for all API interactions (Apply, Delete, Watch):

```sh
go get k8s.io/client-go@latest
go get k8s.io/apimachinery@latest
```

Build the client from the default kubeconfig:

```go
// internal/kubernetes/client.go

import (
    "k8s.io/client-go/dynamic"
    "k8s.io/client-go/tools/clientcmd"
)

func NewDynamicClient() (dynamic.Interface, error) {
    loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
    config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
        loadingRules, &clientcmd.ConfigOverrides{},
    ).ClientConfig()
    if err != nil {
        return nil, err
    }
    return dynamic.NewForConfig(config)
}
```

### Applying manifests (server-side apply)

Use the dynamic client with server-side apply. Parse the YAML into
`k8s.io/apimachinery/pkg/apis/meta/v1/unstructured`, then call `Apply` with a field manager:

```go
import (
    "k8s.io/apimachinery/pkg/api/meta"
    "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
    "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/types"
    "k8s.io/client-go/discovery"
    "k8s.io/client-go/restmapper"
)

// ApplyOp.Run — simplified sketch
func (op ApplyOp) Run(ctx context.Context) error {
    data, err := os.ReadFile(op.Path)
    // ... parse YAML into unstructured.Unstructured (handle multi-doc YAML with "---")
    // ... discover GVR from the object's GroupVersionKind using restmapper
    // ... call dynamicClient.Resource(gvr).Namespace(ns).Apply(ctx, name, obj,
    //         metav1.ApplyOptions{FieldManager: "auto-homelab", Force: true})
}
```

The `restmapper` package maps a `GroupVersionKind` (from the YAML) to a
`GroupVersionResource` (what the dynamic client needs). See
`k8s.io/client-go/restmapper.GetAPIGroupResources` for building the mapper.

For multi-document YAML files (separated by `---`), split on `\n---\n` and apply each document
separately.

### The k8s up command

```go
// cmd/k8s.go (sketch)

func upOps(env string, projectRoot string) []kubernetes.Op {
    base := []kubernetes.Op{
        kubernetes.ApplyOp{Path: "kubernetes/grimmory/namespace.yaml"},
        kubernetes.ApplyOp{Path: "kubernetes/grimmory/configmap.yaml"},
        kubernetes.ApplyOp{Path: "kubernetes/grimmory/secret.yaml"},
        kubernetes.ApplyOp{Path: "kubernetes/grimmory/storage/pvs." + env + ".yaml"},
        kubernetes.ApplyOp{Path: "kubernetes/grimmory/storage/pvcs.yaml"},
        kubernetes.ApplyOp{Path: "kubernetes/grimmory/mariadb/deployment.yaml"},
        kubernetes.ApplyOp{Path: "kubernetes/grimmory/mariadb/service.yaml"},
        kubernetes.ApplyOp{Path: "kubernetes/grimmory/grimmory/deployment.yaml"},
        kubernetes.ApplyOp{Path: "kubernetes/grimmory/grimmory/service.yaml"},
        kubernetes.WaitPodsReadyOp{Namespace: "grimmory", Timeout: 5 * time.Minute},
    }

    if env == "dev" {
        devOps := []kubernetes.Op{
            kubernetes.ExecOp{Cmd: "minikube", Args: []string{"start", "--driver=docker"}},
            kubernetes.EnsureDirOp{Path: filepath.Join(projectRoot, "test-data/grimmory/app-data")},
            kubernetes.EnsureDirOp{Path: filepath.Join(projectRoot, "test-data/grimmory/books")},
            kubernetes.EnsureDirOp{Path: filepath.Join(projectRoot, "test-data/grimmory/bookdrop")},
            kubernetes.EnsureDirOp{Path: filepath.Join(projectRoot, "test-data/grimmory/mariadb")},
            kubernetes.BackgroundExecOp{
                Cmd:     "minikube",
                Args:    []string{"mount", projectRoot + "/test-data:/test-data"},
                PIDFile: "/tmp/minikube-mount-grimmory.pid",
            },
        }
        return append(devOps, base...)
    }
    return base
}
```

Ordering matters: namespace must exist before namespace-scoped resources; PVs must exist before
PVCs; PVCs must be bound before Deployments can schedule pods.

### The k8s configure command (planned)

This command will mirror how `cmd/configure.go` works for Docker Compose:

1. Load `secret.template.yaml` to know which keys are required.
2. Prompt the user for each value (reusing `internal/config.Prompter`).
3. Create or update the Secret via the Kubernetes API (`CoreV1().Secrets().Apply(...)`).

This keeps credentials out of the filesystem entirely — they flow from the user's terminal directly
into the cluster.

### Testing

Each `Op` type is independently testable:

- `EnsureDirOp`: assert the directory exists after `Run`.
- `ExecOp`: inject a fake runner (interface) and assert the right command was called.
- `ApplyOp`: use `envtest` (from `sigs.k8s.io/controller-runtime/pkg/envtest`) to run a real API
  server in tests and assert the resource was created with the correct spec.
- `WaitPodsReadyOp`: fake the watch stream with test fixtures.

This is the advantage over shell scripts: every unit of work is a Go struct with a single `Run`
method that can be unit-tested in isolation.
