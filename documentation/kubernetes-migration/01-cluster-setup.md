# Kubernetes Migration: Cluster Setup (kubeadm on Debian)

This document covers bootstrapping a single-node Kubernetes cluster on Debian using `kubeadm`,
choosing a CNI plugin, and installing the storage class that the rest of this guide depends on.

---

## Prerequisites on the Debian host

```bash
# Disable swap — kubelet refuses to start with swap on
swapoff -a
sed -i '/ swap / s/^/#/' /etc/fstab   # persist across reboots

# Load required kernel modules
cat <<EOF | tee /etc/modules-load.d/k8s.conf
overlay
br_netfilter
EOF
modprobe overlay
modprobe br_netfilter

# Required sysctl settings
cat <<EOF | tee /etc/sysctl.d/k8s.conf
net.bridge.bridge-nf-call-iptables  = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward                 = 1
EOF
sysctl --system
```

---

## Install containerd (replaces Docker as the container runtime)

Kubernetes uses the CRI interface. containerd is the standard runtime for kubeadm clusters.
Note: if Docker is already installed, containerd is already present — but you need to
configure it for CRI use.

```bash
apt-get update
apt-get install -y containerd

# Generate default config and enable SystemdCgroup (required for kubeadm)
mkdir -p /etc/containerd
containerd config default | tee /etc/containerd/config.toml
# Edit the file: find [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
# and set SystemdCgroup = true
sed -i 's/SystemdCgroup = false/SystemdCgroup = true/' /etc/containerd/config.toml

systemctl restart containerd
systemctl enable containerd
```

> **Note on Docker**: The existing Docker installation remains on the machine and continues
> to run your Docker Compose stack during the transition. Kubernetes uses containerd directly
> and does not conflict with Docker.

---

## Install kubeadm, kubelet, kubectl

```bash
apt-get install -y apt-transport-https ca-certificates curl gpg

curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.32/deb/Release.key \
  | gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg

echo 'deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] \
  https://pkgs.k8s.io/core:/stable:/v1.32/deb/ /' \
  | tee /etc/apt/sources.list.d/kubernetes.list

apt-get update
apt-get install -y kubelet kubeadm kubectl
apt-mark hold kubelet kubeadm kubectl   # prevent unattended upgrades
```

---

## Bootstrap the cluster

```bash
# Replace with your server's actual LAN IP
SERVER_IP=192.168.1.100

kubeadm init \
  --pod-network-cidr=10.244.0.0/16 \
  --apiserver-advertise-address=$SERVER_IP \
  --node-name=$(hostname)
```

After it completes:

```bash
mkdir -p $HOME/.kube
cp /etc/kubernetes/admin.conf $HOME/.kube/config
chown $(id -u):$(id -g) $HOME/.kube/config
```

Allow the control-plane node to also run workloads (required on a single-node cluster):

```bash
kubectl taint nodes --all node-role.kubernetes.io/control-plane-
```

---

## Install a CNI plugin (Flannel)

Flannel is the simplest CNI plugin for a single-node homelab. It uses the `10.244.0.0/16`
CIDR specified above.

```bash
kubectl apply -f https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml
```

Alternatives to consider:
- **Calico** — more features (network policies, BGP), more complex
- **Cilium** — eBPF-based, excellent observability, heavier on resources
- **Flannel** — recommended for a single-node homelab; minimal overhead

Verify the node is ready:

```bash
kubectl get nodes
# NAME       STATUS   ROLES           AGE   VERSION
# hostname   Ready    control-plane   2m    v1.32.x
```

---

## Namespace strategy

Rather than putting everything in `default`, use one namespace per concern:

```bash
kubectl create namespace homelab          # all homelab services
kubectl create namespace homelab-system   # infrastructure (Traefik, cert-manager, AdGuard)
```

This mirrors what the current project separates logically — infrastructure vs applications —
and allows you to apply RBAC rules per namespace later.

---

## Install local-path-provisioner (storage class)

See `03-storage.md` for full detail. The short version:

```bash
kubectl apply -f https://raw.githubusercontent.com/rancher/local-path-provisioner/master/deploy/local-path-storage.yaml

# Make it the default storage class
kubectl patch storageclass local-path \
  -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'
```

This gives you dynamic provisioning of `PersistentVolumeClaims` backed by host directories,
which is the closest equivalent to Docker's named volumes on a single node.

---

## Install Helm (package manager)

Several components in this guide (Traefik, cert-manager) are best installed via Helm.

```bash
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
```

---

## Verify the cluster

```bash
kubectl get pods -A
# You should see: coredns, flannel, kube-proxy, local-path-provisioner all Running
```

---

## kubeconfig vs docker-context

The current `--docker-context` flag problem in the Go CLI is replaced by kubeconfig contexts.
The kubeconfig file (default: `~/.kube/config`) supports multiple clusters and contexts. The
Go `client-go` library reads it automatically:

```go
import "k8s.io/client-go/tools/clientcmd"

config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
```

If you ever need to manage a remote cluster, you add its context to your local kubeconfig and
switch with `kubectl config use-context <name>` — or pass `--kubeconfig` / `--context` flags,
which `client-go` supports natively. No custom flag-threading needed.

---

## Resource overhead

On a fresh single-node cluster (Debian, 1 node), expect the control plane + CNI + DNS to
consume approximately:

| Component | CPU (idle) | Memory |
|---|---|---|
| API server | ~50m | ~300 MB |
| etcd | ~20m | ~100 MB |
| scheduler | ~5m | ~50 MB |
| controller-manager | ~10m | ~70 MB |
| CoreDNS (×2) | ~5m | ~60 MB |
| kubelet | ~20m | ~80 MB |
| Flannel | ~5m | ~30 MB |
| **Total overhead** | **~115m** | **~690 MB** |

This is the cost of the control plane on top of your workloads. On a machine with 8+ GB RAM
it is acceptable. On 4 GB it is tight — consider a `k3s` or `k0s` distribution instead, which
have significantly smaller control plane footprints while remaining fully kubeadm-compatible
for workloads.
