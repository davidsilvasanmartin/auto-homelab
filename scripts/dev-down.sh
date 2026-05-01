#!/usr/bin/env bash
# Tear down the Grimmory dev cluster.
# Passes --keep-data to preserve test-data/ by default.
# Pass --wipe to also delete all data in test-data/grimmory/.
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MOUNT_PIDFILE="/tmp/minikube-mount-grimmory.pid"
WIPE=false

for arg in "$@"; do
  [ "$arg" = "--wipe" ] && WIPE=true
done

# ── 1. Remove Kubernetes resources ────────────────────────────────────────────
echo "==> Removing Kubernetes resources..."
kubectl delete namespace grimmory --ignore-not-found
kubectl delete pv grimmory-app-data grimmory-books grimmory-bookdrop grimmory-mariadb-data \
  --ignore-not-found

# ── 2. Stop minikube mount ────────────────────────────────────────────────────
if [ -f "$MOUNT_PIDFILE" ]; then
  echo "==> Stopping minikube mount (PID $(cat "$MOUNT_PIDFILE"))..."
  kill "$(cat "$MOUNT_PIDFILE")" 2>/dev/null || true
  rm "$MOUNT_PIDFILE"
else
  echo "==> No mount pidfile found (mount may have already stopped)"
fi

# ── 3. Stop Minikube ──────────────────────────────────────────────────────────
echo "==> Stopping Minikube..."
minikube stop

# ── 4. Optionally wipe data ───────────────────────────────────────────────────
if [ "$WIPE" = true ]; then
  echo "==> Wiping test-data/grimmory/ ..."
  rm -rf "$PROJECT_ROOT/test-data/grimmory"
  echo "==> Done. Data deleted."
else
  echo "==> Done. test-data/grimmory/ is preserved. Run with --wipe to delete it."
fi
