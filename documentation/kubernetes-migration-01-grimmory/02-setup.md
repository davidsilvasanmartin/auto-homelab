# Grimmory on Kubernetes — Setup

Install the tools and run the startup script. This page only covers what needs to be done once on
a new machine. The script handles everything else.

---

## 1. Install dependencies

Docker Desktop must be installed and **running** — Minikube uses it as its engine.

Install the remaining tools via Homebrew:

```sh
brew install minikube kubernetes-cli
```

Verify:

```sh
minikube version   # v1.x.x
kubectl version --client --short   # v1.x.x
```

---

## 2. Configure Docker Desktop memory

Grimmory is a JVM app and MariaDB also needs headroom. Open Docker Desktop →
**Settings → Resources → Advanced** and set memory to at least **4 GB** (6 GB is comfortable).

---

## 3. Start the dev environment

From the project root:

```sh
scripts/dev-up.sh
```

The script does the following in order:

| Step | What happens |
|---|---|
| Minikube | Starts the cluster with the Docker driver (skips if already running) |
| test-data | Creates the local directories that back the volumes |
| minikube mount | Bridges `./test-data` into the cluster as `/test-data` (runs in background) |
| Manifests | Applies namespace → PVs → PVCs → Secret → Deployments → Services |
| Watch | Streams pod status until you press Ctrl+C |

Once both pods show `1/1 Running`, open the app:

```sh
minikube service grimmory -n grimmory
```

Minikube creates a localhost tunnel and opens your browser.

---

## What the mount process means for restarts

`minikube mount` must stay running for pods to access storage. The script starts it as a background
process and stores its PID in `/tmp/minikube-mount-grimmory.pid`.

- If you **reboot your Mac**, run `scripts/dev-up.sh` again — it detects the stale PID and restarts
  the mount.
- If you **just want to reapply manifests** after changing a YAML file, `scripts/dev-up.sh` is safe
  to re-run; `kubectl apply` is idempotent.

---

## Tearing down

```sh
# Stop the cluster (preserves test-data/)
scripts/dev-down.sh

# Stop the cluster and delete all data
scripts/dev-down.sh --wipe
```

Continue to [03-deploy.md](03-deploy.md) for useful runtime commands and troubleshooting.
