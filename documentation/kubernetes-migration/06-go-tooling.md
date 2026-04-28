# Kubernetes Migration: Go CLI Tooling

This document covers how to migrate the existing Go CLI from Docker Compose shell-outs
to the Kubernetes `client-go` library. The CLI structure (Cobra commands, internal package
layout, interface-based design) can remain largely unchanged — only the transport layer
changes.

---

## What changes and what stays the same

| Component | Today | In Kubernetes |
|---|---|---|
| CLI framework | Cobra + root flags | Cobra + kubeconfig / context flags |
| `--docker-context` flag | Custom flag in root.go | Standard `--kubeconfig` / `--context` |
| `docker compose up` | Shell-out via `ExecShellCommand` | `client-go` Deployment scale/apply |
| `docker compose stop` | Shell-out | Scale Deployment replicas to 0 |
| `docker exec <container> pg_dump` | Shell-out | `client-go` Exec subresource |
| `cp` for file copying | Shell-out via `FilesHandler` | Stays the same (local backup path on host) |
| `restic` for cloud backup | Shell-out via `ResticClient` | Stays the same (CronJob or local call) |
| `.env` config file | Viper-loaded | Reads from / writes to ConfigMap + Secret |
| `configure` command | Writes `.env` file | Writes ConfigMap + Secret objects |

The `LocalBackup` and `ResticClient` interfaces are minimally affected — they still call
`pg_dump`, `mariadb-dump`, and `restic` via exec. Only the exec mechanism changes from
`docker exec` to `kubectl exec` (the `client-go` Exec subresource).

---

## New dependency: client-go

Add the Kubernetes client to the Go module:

```bash
go get k8s.io/client-go@v0.32.0
go get k8s.io/api@v0.32.0
go get k8s.io/apimachinery@v0.32.0
```

The `client-go` version should match the cluster's Kubernetes version (1.32 in this guide).
client-go is backwards compatible within one minor version in either direction.

---

## Replace the Docker runner

### Today: `internal/docker/runner.go`

```go
type SystemRunner struct {
    commands system.Commands
    dockerContext string
    buildDockerComposeCommandStr func(cmd string, ctx string) string
}

func (r *SystemRunner) ComposeStart(services []string) error {
    cmd := r.buildDockerComposeCommandStr("up -d " + strings.Join(services, " "), r.dockerContext)
    return r.commands.ExecShellCommand(cmd).Run()
}

func (r *SystemRunner) ContainerExec(container, cmd string) error {
    dockerCmd := fmt.Sprintf("docker exec %s %s", container, cmd)
    return r.commands.ExecShellCommand(dockerCmd).Run()
}
```

### In Kubernetes: `internal/kube/runner.go`

```go
package kube

import (
    "bytes"
    "context"

    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/kubernetes/scheme"
    "k8s.io/client-go/rest"
    "k8s.io/client-go/tools/clientcmd"
    "k8s.io/client-go/tools/remotecommand"
)

type Runner interface {
    ScaleDeployment(namespace, name string, replicas int32) error
    PodExec(namespace, podName, container string, cmd []string) (stdout, stderr string, err error)
    GetPodForDeployment(namespace, deploymentName string) (string, error)
}

type SystemRunner struct {
    client     kubernetes.Interface
    restConfig *rest.Config
}

func NewSystemRunner(kubeconfig, context string) (*SystemRunner, error) {
    loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
    if kubeconfig != "" {
        loadingRules.ExplicitPath = kubeconfig
    }
    overrides := &clientcmd.ConfigOverrides{}
    if context != "" {
        overrides.CurrentContext = context
    }
    config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
        loadingRules, overrides,
    ).ClientConfig()
    if err != nil {
        return nil, err
    }
    client, err := kubernetes.NewForConfig(config)
    if err != nil {
        return nil, err
    }
    return &SystemRunner{client: client, restConfig: config}, nil
}

// ScaleDeployment starts (replicas > 0) or stops (replicas = 0) a deployment.
func (r *SystemRunner) ScaleDeployment(namespace, name string, replicas int32) error {
    ctx := context.Background()
    scale, err := r.client.AppsV1().Deployments(namespace).GetScale(ctx, name, metav1.GetOptions{})
    if err != nil {
        return err
    }
    scale.Spec.Replicas = replicas
    _, err = r.client.AppsV1().Deployments(namespace).UpdateScale(ctx, name, scale, metav1.UpdateOptions{})
    return err
}

// GetPodForDeployment returns the name of the first running pod for a deployment.
func (r *SystemRunner) GetPodForDeployment(namespace, deploymentName string) (string, error) {
    ctx := context.Background()
    deploy, err := r.client.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
    if err != nil {
        return "", err
    }
    selector := deploy.Spec.Selector.MatchLabels
    labelStr := metav1.FormatLabelSelector(&metav1.LabelSelector{MatchLabels: selector})
    pods, err := r.client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelStr})
    if err != nil {
        return "", err
    }
    for _, pod := range pods.Items {
        if pod.Status.Phase == corev1.PodRunning {
            return pod.Name, nil
        }
    }
    return "", fmt.Errorf("no running pod found for deployment %s/%s", namespace, deploymentName)
}

// PodExec runs a command inside a container of a pod, equivalent to kubectl exec.
func (r *SystemRunner) PodExec(namespace, podName, container string, cmd []string) (string, string, error) {
    req := r.client.CoreV1().RESTClient().Post().
        Resource("pods").
        Name(podName).
        Namespace(namespace).
        SubResource("exec").
        VersionedParams(&corev1.PodExecOptions{
            Container: container,
            Command:   cmd,
            Stdout:    true,
            Stderr:    true,
        }, scheme.ParameterCodec)

    exec, err := remotecommand.NewSPDYExecutor(r.restConfig, "POST", req.URL())
    if err != nil {
        return "", "", err
    }

    var stdout, stderr bytes.Buffer
    err = exec.StreamWithContext(context.Background(), remotecommand.StreamOptions{
        Stdout: &stdout,
        Stderr: &stderr,
    })
    return stdout.String(), stderr.String(), err
}
```

---

## Start and stop commands

The `start` / `stop` commands today call `docker compose up -d` and `docker compose stop`.
In Kubernetes, "start" means scaling a Deployment to its desired replicas, and "stop" means
scaling to zero.

```go
// cmd/start.go
func runStart(services []string) error {
    runner, err := kube.NewSystemRunner(kubeconfig, kubeContext)
    if err != nil {
        return err
    }
    if len(services) == 0 {
        services = allServices // defined list of deployment names
    }
    for _, svc := range services {
        if err := runner.ScaleDeployment("homelab", svc, 1); err != nil {
            return fmt.Errorf("starting %s: %w", svc, err)
        }
    }
    return nil
}

// cmd/stop.go
func runStop(services []string) error {
    // same pattern, replicas = 0
    return runner.ScaleDeployment("homelab", svc, 0)
}
```

> **Note**: StatefulSets (databases) should generally not be scaled to zero independently —
> stopping a database without stopping its dependent application first will cause crashes.
> The stop command should enforce ordering: stop app first, then its database.

---

## Database exec for backups

The current `PostgreSQLLocalBackup` calls:

```go
docker exec <container> pg_dump -U <user> <db>
```

In Kubernetes this becomes:

```go
type PostgreSQLLocalBackup struct {
    runner    kube.Runner
    namespace string
    deployment string
    container  string
    dbName, username, password string
    dstPath string
}

func (b *PostgreSQLLocalBackup) Run() error {
    podName, err := b.runner.GetPodForDeployment(b.namespace, b.deployment)
    if err != nil {
        return err
    }
    cmd := []string{
        "pg_dump",
        "-U", b.username,
        "--no-password",
        b.dbName,
    }
    stdout, stderr, err := b.runner.PodExec(b.namespace, podName, b.container, cmd)
    if err != nil {
        return fmt.Errorf("pg_dump failed: %w\nstderr: %s", err, stderr)
    }
    return os.WriteFile(b.dstPath, []byte(stdout), 0600)
}
```

The same pattern applies to `MariaDBLocalBackup` and `MySQLLocalBackup`, only changing
the command from `pg_dump` to `mariadb-dump`.

---

## The configure command: writing ConfigMaps and Secrets

The current `configure` command writes a `.env` file. The Kubernetes equivalent writes
`ConfigMap` and `Secret` objects.

The `Configurer` interface can stay the same — only the `WriteConfig` implementation changes:

```go
// internal/config/kube_writer.go

type KubeConfigWriter struct {
    client    kubernetes.Interface
    namespace string
}

func (w *KubeConfigWriter) WriteConfig(root *EnvVarRoot) error {
    configData := map[string]string{}
    secretData := map[string][]byte{}

    for _, section := range root.Sections {
        for _, v := range section.Vars {
            if v.Sensitive {
                secretData[v.Name] = []byte(v.Value)
            } else {
                configData[v.Name] = v.Value
            }
        }
    }

    ctx := context.Background()

    cm := &corev1.ConfigMap{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "homelab-config",
            Namespace: w.namespace,
        },
        Data: configData,
    }
    _, err := w.client.CoreV1().ConfigMaps(w.namespace).Apply(ctx, cm, ...)
    if err != nil {
        return err
    }

    secret := &corev1.Secret{
        ObjectMeta: metav1.ObjectMeta{Name: "homelab-secrets", Namespace: w.namespace},
        Data: secretData,
    }
    _, err = w.client.CoreV1().Secrets(w.namespace).Apply(ctx, secret, ...)
    return err
}
```

The `env.config.json` schema needs a new `sensitive: true` field per variable to distinguish
ConfigMap entries from Secret entries. Generated passwords, API tokens, and database passwords
are `sensitive: true`; domain names, timezone, IPs are not.

---

## Root command: kubeconfig and context flags

Replace the `--docker-context` flag with standard Kubernetes flags:

```go
// cmd/root.go
var (
    kubeconfig string
    kubeContext string
)

func init() {
    rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "",
        "Path to kubeconfig (defaults to ~/.kube/config)")
    rootCmd.PersistentFlags().StringVar(&kubeContext, "context", "",
        "Kubernetes context to use")
}
```

This matches the convention of every other Kubernetes-aware CLI tool. The Go `client-go`
library reads these flags natively via `clientcmd`.

---

## WaitUntilContainerExecIsSuccessful

The current implementation retries a `docker exec` command with a 1-second sleep up to
30 times. The Kubernetes equivalent is structurally identical — just use `PodExec` instead:

```go
func (r *SystemRunner) WaitUntilPodExecSucceeds(namespace, deployment, container string,
    cmd []string, maxAttempts int, interval time.Duration) error {

    for i := 0; i < maxAttempts; i++ {
        podName, err := r.GetPodForDeployment(namespace, deployment)
        if err == nil {
            _, _, err = r.PodExec(namespace, podName, container, cmd)
            if err == nil {
                return nil
            }
        }
        time.Sleep(interval)
    }
    return fmt.Errorf("command did not succeed after %d attempts", maxAttempts)
}
```

In practice, `initContainers` in the pod spec handle most of this waiting — the Go CLI only
needs this for cases where it must wait from outside the cluster (e.g. in the backup command,
waiting for a database to be ready before running `pg_dump`).

---

## FilesHandler

The `internal/system/files.go` handler uses the system `cp` command. This does not need to
change for backup purposes — it still copies from host paths (where local-path-provisioner
creates directories) to the backup path on the same host. The paths are different (no more
`.env` vars pointing to arbitrary directories; instead you find the PV backing directory),
but the mechanism is the same.

---

## Summary of interface changes

| Interface | Changes |
|---|---|
| `Runner` | Rename to `kube.Runner`, replace `ComposeStart/Stop` with `ScaleDeployment`, replace `ContainerExec` with `PodExec` |
| `LocalBackup` | No interface change — `Run()` still returns `error` |
| `PostgreSQLLocalBackup` | Replaces `docker exec` with `PodExec` |
| `CloudBackup` / `ResticClient` | No change — still shells out to `restic` |
| `Configurer` | Add `KubeConfigWriter` alongside existing `EnvFileWriter` |
| `FilesHandler` | No change — still copies host directories |
| `system.Commands` | No change — still wraps `os/exec` |
