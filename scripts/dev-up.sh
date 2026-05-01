#!/usr/bin/env bash
# Bring up the Grimmory dev cluster on Minikube (Docker driver).
# Safe to run repeatedly — kubectl apply and helm are idempotent.
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
K8S="$PROJECT_ROOT/kubernetes/grimmory"
MOUNT_PIDFILE="/tmp/minikube-mount-grimmory.pid"
MOUNT_LOG="/tmp/minikube-mount-grimmory.log"

# ── 1. Minikube ───────────────────────────────────────────────────────────────
if minikube status -f '{{.Host}}' 2>/dev/null | grep -q Running; then
  echo "==> Minikube already running"
else
  echo "==> Starting Minikube (docker driver)..."
  minikube start --driver=docker
fi

# ── 2. test-data directories ──────────────────────────────────────────────────
echo "==> Ensuring test-data directories exist..."
mkdir -p "$PROJECT_ROOT/test-data/grimmory/app-data" \
         "$PROJECT_ROOT/test-data/grimmory/books" \
         "$PROJECT_ROOT/test-data/grimmory/bookdrop" \
         "$PROJECT_ROOT/test-data/grimmory/mariadb"

# ── 3. minikube mount ─────────────────────────────────────────────────────────
if [ -f "$MOUNT_PIDFILE" ] && kill -0 "$(cat "$MOUNT_PIDFILE")" 2>/dev/null; then
  echo "==> Minikube mount already running (PID $(cat "$MOUNT_PIDFILE"))"
else
  echo "==> Starting minikube mount in background..."
  minikube mount "$PROJECT_ROOT/test-data:/test-data" >"$MOUNT_LOG" 2>&1 &
  echo $! >"$MOUNT_PIDFILE"
  sleep 2   # give the 9p server a moment to start
  echo "==> Mount running (PID $(cat "$MOUNT_PIDFILE"), log: $MOUNT_LOG)"
fi

# ── 4. Kubernetes manifests ───────────────────────────────────────────────────
echo "==> Applying manifests..."
kubectl apply -f "$K8S/namespace.yaml"
kubectl apply -f "$K8S/storage/pvs.yaml"
kubectl apply -f "$K8S/storage/pvcs.yaml"
kubectl apply -f "$K8S/secret.yaml"
kubectl apply -f "$K8S/mariadb/deployment.yaml"
kubectl apply -f "$K8S/mariadb/service.yaml"
kubectl apply -f "$K8S/grimmory/deployment.yaml"
kubectl apply -f "$K8S/grimmory/service.yaml"

# ── 5. Done ───────────────────────────────────────────────────────────────────
echo ""
echo "==> Cluster is up. Watching pods (Ctrl+C to stop watching):"
echo "    Once both pods show 1/1 Running, open the app with:"
echo "      minikube service grimmory -n grimmory"
echo ""
kubectl get pods -n grimmory --watch
