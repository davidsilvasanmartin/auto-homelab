# Kubernetes Migration: Networking

This document covers how the current Docker network model maps to Kubernetes, how Traefik
works as an Ingress controller, and how to solve the AdGuard port-53 problem.

---

## How the Docker network model maps to Kubernetes

### Today

Every service gets a static IP on the `homelab_network` bridge network (e.g.
`HOMELAB_ADGUARD_IP`, `HOMELAB_TRAEFIK_IP`). Services reach each other by IP. The static
IPs matter because AdGuard's configuration is written with the exact IPs of upstream services.

```yaml
networks:
  homelab_network:
    ipam:
      config:
        - subnet: ${HOMELAB_GENERAL_DOCKER_NETWORK}  # 10.42.42.0/24

services:
  paperless-db:
    networks:
      homelab_network:
        ipv4_address: ${HOMELAB_PAPERLESS_DB_IP}    # e.g. 10.42.42.30
```

### In Kubernetes

You do not assign static IPs to pods — pods are ephemeral and their IPs change. Instead,
you create a `Service` object that gives a stable DNS name and virtual IP (ClusterIP) to a
set of pods. The DNS name is:

```
<service-name>.<namespace>.svc.cluster.local
```

Or within the same namespace simply `<service-name>`.

```yaml
# paperless-db service — stable DNS name within the cluster
apiVersion: v1
kind: Service
metadata:
  name: paperless-db
  namespace: homelab
spec:
  selector:
    app: paperless-db
  ports:
    - port: 5432
      targetPort: 5432
  type: ClusterIP
```

Paperless then connects to `paperless-db:5432` instead of `10.42.42.30:5432`. This is
cleaner than static IPs.

### What about AdGuard's rewrite rules?

AdGuard's config today likely contains entries like:

```yaml
rewrites:
  - domain: "immich.home.example.com"
    answer: "10.42.42.X"   # Traefik's static IP
```

In Kubernetes, these rewrites should all point to the **node's IP** (or the LoadBalancer IP
if using MetalLB), since Traefik runs on the node and handles routing from there. If the node
IP is static (which it typically is for a homelab server), this is straightforward.

---

## Traefik as an Ingress Controller

### Today

Traefik is a compose service. Other services add Docker labels to opt in to routing:

```yaml
labels:
  - "traefik.http.routers.immich.rule=Host(`immich.home.example.com`)"
  - "traefik.http.services.immich.loadbalancer.server.port=2283"
```

Traefik discovers these labels via the Docker socket.

### In Kubernetes

Traefik becomes a proper Ingress controller. It watches `Ingress` objects (or its own
`IngressRoute` CRDs) and routes accordingly. The Docker label mechanism is replaced by
Kubernetes resources.

#### Install Traefik via Helm

```bash
helm repo add traefik https://traefik.github.io/charts
helm repo update

helm install traefik traefik/traefik \
  --namespace homelab-system \
  --create-namespace \
  --set ports.web.hostPort=80 \
  --set ports.websecure.hostPort=443 \
  --set service.type=ClusterIP
```

Setting `hostPort` binds Traefik directly to ports 80 and 443 on the node, equivalent to
the current compose `ports: ["80:80", "443:443"]`. On a single-node homelab this is the
simplest approach.

#### Ingress resource (standard)

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: immich
  namespace: homelab
  annotations:
    traefik.ingress.kubernetes.io/router.entrypoints: websecure
    traefik.ingress.kubernetes.io/router.tls: "true"
    traefik.ingress.kubernetes.io/router.tls.certresolver: cloudflare
spec:
  ingressClassName: traefik
  rules:
    - host: immich.home.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: immich
                port:
                  number: 2283
```

#### IngressRoute CRD (Traefik-native, more expressive)

```yaml
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: immich
  namespace: homelab
spec:
  entryPoints:
    - websecure
  routes:
    - match: Host(`immich.home.example.com`)
      kind: Rule
      services:
        - name: immich
          port: 2283
  tls:
    certResolver: cloudflare
```

The `IngressRoute` approach maps more naturally from the current Traefik compose labels.

---

## Solving the AdGuard port-53 problem

This is the most complex networking challenge in the migration. Port 53 is below the NodePort
range (30000–32767), so a standard `NodePort` service cannot expose it.

### Option A: `hostNetwork: true` (simplest)

The AdGuard pod runs in the host's network namespace. Port 53 on the node is AdGuard's port 53.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: adguard
  namespace: homelab-system
spec:
  template:
    spec:
      hostNetwork: true        # shares host network namespace
      dnsPolicy: ClusterFirstWithHostNet   # still use cluster DNS for pod lookups
      containers:
        - name: adguard
          image: adguard/adguardhome:with-apache2-utils
          ports:
            - containerPort: 53
              protocol: UDP
            - containerPort: 53
              protocol: TCP
            - containerPort: 3000
```

**Pros**: Works immediately, no extra components.
**Cons**: The pod can bind to any port on the host. Only one AdGuard pod can run per node
(since it physically owns port 53). Not portable to multi-node clusters.

### Option B: `hostPort` on the container (slightly narrower)

```yaml
containers:
  - name: adguard
    ports:
      - containerPort: 53
        hostPort: 53
        protocol: UDP
      - containerPort: 53
        hostPort: 53
        protocol: TCP
```

**Pros**: More explicit than `hostNetwork`.
**Cons**: Still pins the pod to a specific node. The Kubernetes project discourages `hostPort`
for most use cases.

### Option C: MetalLB (cleanest, most production-like)

MetalLB implements a `LoadBalancer` service type for bare-metal clusters. It assigns a real
IP from a pool you configure (e.g. `192.168.1.200`) to the service. AdGuard gets its own IP
separate from the node IP, with proper port 53 on both TCP and UDP.

```bash
kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.14.8/config/manifests/metallb-native.yaml
```

Configure an IP pool:

```yaml
apiVersion: metallb.io/v1beta1
kind: IPAddressPool
metadata:
  name: homelab-pool
  namespace: metallb-system
spec:
  addresses:
    - 192.168.1.200-192.168.1.210   # unused IPs on your LAN
---
apiVersion: metallb.io/v1beta1
kind: L2Advertisement
metadata:
  name: homelab
  namespace: metallb-system
```

Then the AdGuard service becomes:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: adguard-dns
  namespace: homelab-system
spec:
  type: LoadBalancer
  loadBalancerIP: 192.168.1.200
  selector:
    app: adguard
  ports:
    - name: dns-udp
      port: 53
      protocol: UDP
    - name: dns-tcp
      port: 53
      protocol: TCP
```

Point your router's DNS at `192.168.1.200`. This is the cleanest separation and works
correctly in multi-node scenarios.

**Recommended**: Option A for simplicity now, Option C if you expand to multiple nodes or
want cleaner isolation.

---

## CoreDNS conflict

Kubernetes ships with CoreDNS listening on port 53 **inside the cluster** (on a ClusterIP
address). This does not conflict with AdGuard on the host network, because CoreDNS only
listens on its ClusterIP (e.g. `10.96.0.10`), not on the host's NIC.

However, with `hostNetwork: true`, AdGuard can see CoreDNS's port — be careful not to
configure AdGuard to forward to `127.0.0.1:53`, as that would create a loop.

---

## Service-to-service communication

Pods within the cluster communicate via service names. The table below shows how each
inter-service connection changes:

| Service | Connects to | Today | In Kubernetes |
|---|---|---|---|
| Paperless | paperless-db | `${HOMELAB_PAPERLESS_DB_IP}` | `paperless-db` (same namespace) |
| Paperless | paperless-redis | container name | `paperless-redis` |
| Immich | immich-db | `${HOMELAB_IMMICH_DB_IP}` | `immich-db` |
| Immich | immich-redis | container name | `immich-redis` |
| Firefly | firefly-db | `${HOMELAB_FIREFLY_DB_IP}` | `firefly-db` |
| All apps | Traefik | via Docker labels | `Ingress` / `IngressRoute` objects |

The pattern is consistent: static IPs become DNS names matching the service name.

---

## Istio: service mesh

### What it is and licence

Istio is a free, open-source service mesh (Apache 2.0). It is a CNCF graduated project,
meaning it is considered production-ready by the cloud-native community. A service mesh sits
between pods and handles all service-to-service networking — encryption, retries, circuit
breaking, routing, and observability — transparently, without changing application code.

It works by injecting an Envoy sidecar proxy into every pod automatically (you opt namespaces
in with a label). This is the extra container per pod you mentioned you're comfortable with.

```
Pod (before Istio)        Pod (after Istio)
┌──────────────┐          ┌────────────────────────────┐
│   paperless  │          │  paperless  │  envoy-proxy │
└──────────────┘          └────────────────────────────┘
```

---

### What Istio would solve

#### Automatic mTLS between all services

Today, traffic between pods (e.g. Paperless → paperless-db) is plain TCP inside the cluster.
Any pod on the cluster can, in principle, connect to any other pod. Istio automatically wraps
all service-to-service traffic in mutual TLS — both sides present certificates, both sides
verify them. This happens with zero application changes.

This is the strongest argument for Istio at homelab scale: your databases are no longer
reachable with a plain TCP connection even from within the cluster.

#### `AuthorizationPolicy` — network segmentation without iptables rules

Istio lets you declare which services are allowed to talk to which others:

```yaml
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: paperless-db-policy
  namespace: homelab
spec:
  selector:
    matchLabels:
      app: paperless-db
  action: ALLOW
  rules:
    - from:
        - source:
            principals:
              - "cluster.local/ns/homelab/sa/paperless"  # only paperless can connect
```

This is a significant improvement over the current Docker bridge network where all containers
on `homelab_network` can reach all other containers freely.

#### Observability out of the box

Istio ships with integrations for Kiali (service topology dashboard), Prometheus (metrics),
Jaeger (distributed tracing), and Grafana. You get a live graph of which services are talking
to which, latency percentiles, error rates, and request volumes — all without adding
instrumentation to your application code.

For debugging "why is Immich slow" or "is Paperless actually hitting the database", this is
genuinely useful even in a homelab.

#### Traffic management: retries and timeouts

Istio `VirtualService` objects let you configure retry behaviour at the mesh level:

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: paperless
  namespace: homelab
spec:
  hosts:
    - paperless
  http:
    - retries:
        attempts: 3
        perTryTimeout: 10s
      timeout: 30s
```

Less relevant for a homelab than for production, but it can help with services that are
slow to become ready after a restart (currently handled by `WaitUntilContainerExecIsSuccessful`
in the Go CLI and `initContainers` in pod specs).

#### Istio Gateway as an alternative to Traefik

Istio ships its own ingress gateway that can fully replace Traefik. It handles TLS
termination and virtual-host routing. However, Traefik is more ergonomic for this project
(familiar, well-documented, better Cloudflare DNS integration), so there is no reason to
swap it out. Traefik and Istio coexist comfortably — Traefik handles external traffic into
the cluster; Istio handles internal traffic between services.

---

### What Istio would NOT solve

- **The port-53 problem**: Istio has no answer for this. AdGuard still needs `hostNetwork`,
  `hostPort`, or MetalLB. The Envoy sidecar is not injected into pods using `hostNetwork`
  (Istio detects this and skips injection), so AdGuard is simply outside the mesh entirely.
  That is actually fine — AdGuard does not need mTLS or Istio observability.

- **The storage complexity**: Istio is a networking layer. PVCs, StatefulSets, and
  local-path-provisioner are unchanged.

- **The backup problem**: Istio does not affect CronJobs or restic.

- **The Go CLI rewrite**: `client-go` is still needed. Istio does add its own CRDs
  (`VirtualService`, `AuthorizationPolicy`, etc.) that the CLI could optionally manage,
  but this is optional.

---

### New challenges Istio introduces

#### Resource overhead

Istio has two parts: the control plane (`istiod`) and the per-pod Envoy sidecars.

| Component | CPU (idle) | Memory |
|---|---|---|
| `istiod` (control plane) | ~50m | ~300–500 MB |
| Envoy sidecar (per pod) | ~10–20m | ~50–100 MB |
| **Per 10 pods** | **+150–200m** | **+500–1000 MB** |

Your homelab has roughly 15–20 pods once all services are running. Expect ~1–1.5 GB of
additional memory consumption on top of the Kubernetes control plane overhead from
`01-cluster-setup.md`. On a powerful machine this is fine; on anything under 16 GB RAM
it is worth measuring first.

#### Envoy does not proxy UDP

This is the most concrete technical limitation. Envoy (the proxy Istio injects) handles
TCP and HTTP. It does not proxy UDP traffic. This means:

- **AdGuard DNS** (UDP port 53): must be excluded from the mesh. As mentioned above, Istio
  automatically skips injection for `hostNetwork` pods, so AdGuard is unaffected.
- **Redis/Valkey**: uses TCP — works fine.
- **All databases**: TCP — works fine.
- No UDP-based application in this stack is affected, but it is worth knowing as a general
  constraint if you add UDP services later.

#### Protocol detection for databases

Istio performs automatic protocol detection on connections. For PostgreSQL, MariaDB, and
Redis, it should detect TCP correctly. However, in some configurations Istio misidentifies
database traffic as HTTP and applies HTTP-specific policies. The fix is to explicitly declare
the protocol on the Service port:

```yaml
# paperless-db service with explicit protocol declaration
spec:
  ports:
    - port: 5432
      targetPort: 5432
      name: postgres    # Istio reads the name: prefix "tcp-" forces TCP mode
                        # or use appProtocol: tcp (preferred in newer Istio versions)
      appProtocol: tcp
```

Do this for every database service (PostgreSQL, MariaDB, Redis, Valkey) to avoid
misdetection.

#### Complexity and debugging difficulty

When something goes wrong in a mesh, diagnosing it is harder than without one. A connection
failure might be a pod bug, a Kubernetes NetworkPolicy, an Istio `AuthorizationPolicy`, a
certificate issue in the mesh, or an Envoy config propagation lag. `istioctl` provides
diagnostic tools (`istioctl analyze`, `istioctl proxy-status`, `istioctl proxy-config`) but
they require familiarity to use effectively.

Istio also adds its own certificates (separate from cert-manager's certificates for external
TLS). The two do not conflict — cert-manager issues certificates for external HTTPS via
Traefik; Istio issues internal mesh certificates via its own CA — but understanding the
difference matters for troubleshooting.

#### `initContainers` conflict

Istio injects its own `initContainer` (`istio-init`) into every pod to set up iptables
rules that redirect traffic through Envoy. This runs before your own `initContainers`. In
most cases this is harmless, but if an `initContainer` tries to make a network connection
that must go through Envoy (e.g. waiting for a database), it will fail because Envoy is not
yet running when `initContainers` execute.

The workaround is `holdApplicationUntilProxyStarts: true` in Istio's config, which delays
the main containers until Envoy is ready. For `initContainers`, use Istio's
`traffic.sidecar.istio.io/excludeOutboundPorts` annotation to exclude the ports your init
containers connect on, so they bypass Envoy entirely.

---

### Lighter alternative: Linkerd

If the main appeal of Istio is automatic mTLS and basic observability, consider Linkerd
instead. Linkerd is also free, open source (Apache 2.0), and CNCF graduated. It offers:

- Automatic mTLS between all pods
- A clean dashboard (similar to Kiali, built-in)
- Prometheus metrics and Grafana dashboards
- Much lower resource usage (~200 MB control plane, ~15–25 MB sidecar per pod)
- Simpler mental model — fewer CRDs, less configuration surface

Linkerd does not offer `AuthorizationPolicy` (it has a simpler `Server` + `HTTPRoute`
model), and its traffic management features are less powerful than Istio's. But for a
homelab where the goals are mTLS and observability, Linkerd reaches those goals at roughly
one-third the resource cost.

---

### Recommendation

| Goal | Recommendation |
|---|---|
| mTLS + observability, willing to invest in learning | Istio |
| mTLS + observability, want lower overhead | Linkerd |
| Just want working services with good routing | Skip the mesh entirely, rely on Kubernetes `NetworkPolicy` for segmentation |

For this homelab specifically: the services are trusted (they're all yours) and they're on
a single node. The primary networking risks are already mitigated by being on a private LAN.
Istio or Linkerd are worth adding if you want to learn mesh concepts, or if you want the
observability dashboards (genuinely useful for a homelab). They are not necessary for
security at this scale.

If you do add Istio, start with it disabled (`istio-injection: disabled` on all namespaces)
and enable it namespace by namespace, verifying that each service group works correctly
before moving on. Do not enable it on `homelab-system` (where Traefik lives) until you are
confident it does not interfere with the ingress gateway.

---

## Tutorial: AdGuard-free networking with Unifi DNS, Traefik, and Istio

This tutorial brings together everything in this document into a concrete, working
configuration. It assumes:

- AdGuard is removed entirely
- DNS for the local network is handled by your Unifi appliance
- Istio is your service mesh
- Traefik is your ingress controller
- cert-manager issues TLS certificates via Let's Encrypt DNS-01 + Cloudflare
- Homelab services are accessible at `*.homelab.davidsilva.dev`
- Public services (e.g. `blog.davidsilva.dev`) live directly under `davidsilva.dev` and are hosted in the cloud

### Architecture overview

```
                         ┌─────────────────────────────────────────────────┐
                         │                  INTERNET                        │
                         │                                                  │
                         │   Cloudflare DNS (authoritative)                 │
                         │   *.homelab.davidsilva.dev → <your public IP>   │
                         │   blog.davidsilva.dev  →  <cloud provider IP>   │
                         └────────────────┬────────────────────────────────┘
                                          │
                              ┌───────────▼───────────┐
                              │    Unifi Firewall      │
                              │                        │
                              │  Port forward:         │
                              │  80/443 → 192.168.1.100│
                              │                        │
                              │  Local DNS wildcard:   │
                              │  *.homelab.davidsilva  │
                              │  .dev → 192.168.1.100  │
                              └───────────┬────────────┘
                                          │  LAN: 192.168.1.0/24
                         ┌────────────────▼────────────────────────────────┐
                         │              Debian server (192.168.1.100)       │
                         │                                                  │
                         │  ┌──────────────────────────────────────────┐   │
                         │  │  Kubernetes cluster                       │   │
                         │  │                                           │   │
                         │  │  Traefik (hostPort 80/443)               │   │
                         │  │     ↓ routes by Host header              │   │
                         │  │  ┌────────────┐  ┌──────────────────┐   │   │
                         │  │  │  immich    │  │    paperless     │   │   │
                         │  │  │  pod       │  │    pod           │   │   │
                         │  │  └─────┬──────┘  └────────┬─────────┘   │   │
                         │  │        │  Istio mTLS       │             │   │
                         │  │  ┌─────▼──────────────────▼─────────┐   │   │
                         │  │  │  immich-db  paperless-db  redis   │   │   │
                         │  │  └───────────────────────────────────┘   │   │
                         │  └──────────────────────────────────────────┘   │
                         └─────────────────────────────────────────────────┘
```

---

### Understanding split-horizon DNS

Split-horizon DNS (also called split-brain DNS) means the same domain name resolves to
different IP addresses depending on where the query originates.

For `immich.homelab.davidsilva.dev`:

| Who is asking | DNS server used | Answer | Where traffic goes |
|---|---|---|---|
| Your phone on the LAN | Unifi (wildcard override) | `192.168.1.100` | Directly to Traefik on the server |
| Your laptop on the LAN | Unifi (wildcard override) | `192.168.1.100` | Directly to Traefik on the server |
| Your phone on mobile data | Cloudflare (public) | `<your public IP>` | Through the internet, port-forwarded by Unifi to Traefik |

From the LAN, traffic never leaves your network. From outside, it goes through Cloudflare
and your router's port forward. In both cases, it arrives at Traefik, which routes it to
the same pod. The TLS certificate is valid in both cases because it was issued for the
real public domain `davidsilva.dev` by Let's Encrypt.

---

### The `blog.davidsilva.dev` case: a public service on the same domain

`blog.davidsilva.dev` is hosted somewhere in the cloud — say, on Netlify or a VPS. Its
Cloudflare DNS record points to the cloud provider's IP, not your homelab.

Because all homelab services live under `*.homelab.davidsilva.dev` and the blog lives
directly under `davidsilva.dev`, there is no collision. The Unifi wildcard
`*.homelab.davidsilva.dev → 192.168.1.100` does not match `blog.davidsilva.dev` — the
two subdomains occupy completely separate zones.

When a LAN client queries `blog.davidsilva.dev`, Unifi has no matching local override and
forwards the query upstream to Cloudflare, which returns the cloud provider's IP. The
homelab is not involved at all. No special exceptions or per-service entries are needed.

---

### Step-by-step configuration

#### Step 1: Remove AdGuard

Stop the AdGuard service in Docker Compose and remove it from `docker-compose.yml` (or, in
Kubernetes, simply do not deploy it). There are no other services that depend on AdGuard.

Confirm your clients can still resolve DNS — they should now be using your Unifi appliance's
DNS server directly (which they always were, unless you explicitly pointed clients at the
AdGuard IP).

#### Step 2: Configure local DNS overrides in Unifi

In the Unifi Network application, navigate to Settings → Networks (or Settings → DNS,
depending on firmware). Add a local DNS record for each homelab service pointing to your
server's LAN IP (`192.168.1.100` in these examples).

Add a single wildcard record pointing the entire `homelab.davidsilva.dev` zone at the
server. All current and future homelab services are covered automatically — no record
needed per service.

If your Unifi firmware does not expose a DNS records UI, configure dnsmasq directly over
SSH on the device. Add a single entry to `/etc/dnsmasq.d/homelab.conf`:

```
address=/.homelab.davidsilva.dev/192.168.1.100
```

The leading dot in `/.homelab.davidsilva.dev/` is dnsmasq syntax for "this domain and all
subdomains". Then restart dnsmasq: `killall -HUP dnsmasq`.

> **Warning**: manual SSH configuration on Unifi devices may be overwritten by firmware
> upgrades. Prefer the UI if it supports DNS records on your firmware version.

Verify from a LAN device:

```bash
dig immich.homelab.davidsilva.dev
# Should return 192.168.1.100

dig anything.homelab.davidsilva.dev
# Should also return 192.168.1.100

dig blog.davidsilva.dev
# Should return the cloud provider's IP — homelab wildcard does not match this
```

#### Step 3: Install Traefik

See the Traefik section earlier in this document. The key configuration for a server without
MetalLB is `hostPort` binding:

```bash
helm install traefik traefik/traefik \
  --namespace homelab-system \
  --create-namespace \
  --set ports.web.hostPort=80 \
  --set ports.websecure.hostPort=443 \
  --set service.type=ClusterIP
```

Traefik now listens on ports 80 and 443 of the physical node (`192.168.1.100`).

#### Step 4: Install cert-manager and configure the ClusterIssuer

Follow `05-tls.md` in full. The result is a wildcard certificate for
`*.homelab.davidsilva.dev` stored as a Kubernetes Secret, renewed automatically by
cert-manager via Cloudflare DNS-01.

This certificate is what makes the browser show the green padlock — it is publicly trusted
because it was issued by Let's Encrypt against the real `davidsilva.dev` domain.

#### Step 5: Install Istio

```bash
# Download istioctl
curl -L https://istio.io/downloadIstio | sh -
export PATH="$PATH:$PWD/istio-*/bin"

# Install with the default profile (includes istiod and ingress gateway)
# We disable the Istio ingress gateway since Traefik handles ingress
istioctl install --set profile=minimal -y
```

The `minimal` profile installs only `istiod` (the control plane). Traefik remains the
ingress gateway — Istio's own gateway is not needed.

Enable sidecar injection for the `homelab` namespace, but not for `homelab-system` (where
Traefik runs — it doesn't need to be in the mesh):

```bash
kubectl label namespace homelab istio-injection=enabled
# homelab-system intentionally excluded
```

#### Step 6: Deploy services

Deploy each service with its `Deployment`, `Service`, and `IngressRoute` manifests as
described in `04-services.md`. When pods in the `homelab` namespace start, Istio
automatically injects the Envoy sidecar.

Add `appProtocol: tcp` to all database `Service` definitions to prevent Istio from
misidentifying database traffic as HTTP:

```yaml
# Example: paperless-db service
spec:
  ports:
    - port: 5432
      targetPort: 5432
      appProtocol: tcp
```

#### Step 7: Configure Traefik to use the wildcard certificate

In `traefik.yaml` (or the Helm values), set the wildcard Secret as the default TLS
certificate so every `IngressRoute` uses it without needing to specify `tls.secretName`
per route:

```yaml
# traefik.yaml static config
tls:
  stores:
    default:
      defaultCertificate:
        secretName: homelab-dot-davidsilva-dev-tls
```

Each `IngressRoute` then just needs:

```yaml
tls: {}   # uses the default store — no certresolver, no secretName needed
```

#### Step 8: Configure port forwarding on Unifi (for external access)

If you want services accessible from outside your LAN, add port forwarding rules in
Unifi (Network → Firewall & Security → Port Forwarding):

```
External port 80  →  192.168.1.100:80
External port 443 →  192.168.1.100:443
```

Services you want publicly accessible need a Cloudflare DNS A record (or a wildcard
`*.homelab.davidsilva.dev` record) pointing to your public IP. Services you want LAN-only
(e.g. `portainer.homelab.davidsilva.dev`, `traefik.homelab.davidsilva.dev`) should have
no Cloudflare record — the Unifi wildcard resolves them locally, but they are unreachable
from outside because Cloudflare has no record to route external traffic to your IP.

---

### Request flow walkthrough

#### Case 1: LAN client accessing `immich.homelab.davidsilva.dev`

```
1. Browser: GET https://immich.homelab.davidsilva.dev
2. OS DNS resolver → Unifi DNS
3. Unifi: wildcard *.homelab.davidsilva.dev matches → returns 192.168.1.100
4. Browser: TCP connect to 192.168.1.100:443
5. Traefik: TLS handshake using *.homelab.davidsilva.dev cert (Let's Encrypt — browser trusts it)
6. Traefik: reads Host header "immich.homelab.davidsilva.dev" → routes to immich Service
7. Traefik connects to the immich ClusterIP (homelab-system excluded from Istio mesh)
8. Istio Envoy sidecar (in immich pod): receives connection, mTLS verified with immich-db
9. Immich serves the response
```

Traffic never leaves the LAN. The cert is publicly trusted. Istio encrypts the
Traefik→immich leg if you label `homelab-system` for injection, but for simplicity it
is left out — Traefik to pod traffic stays inside the kernel's network stack on a single
node anyway.

#### Case 2: External client accessing `immich.homelab.davidsilva.dev`

```
1. Browser: GET https://immich.homelab.davidsilva.dev
2. OS DNS resolver → ISP DNS → Cloudflare
3. Cloudflare: A record for *.homelab.davidsilva.dev → <your public IP>
4. Browser: TCP connect to <public IP>:443
5. Unifi: port-forward rule → forwards to 192.168.1.100:443
6. Traefik: TLS handshake (same cert) → routes to immich Service
7–9. Same as Case 1 from step 8 onward
```

#### Case 3: LAN client accessing `blog.davidsilva.dev` (cloud-hosted)

```
1. Browser: GET https://blog.davidsilva.dev
2. OS DNS resolver → Unifi DNS
3. Unifi: *.homelab.davidsilva.dev wildcard does NOT match blog.davidsilva.dev
         → forwards to upstream (Cloudflare)
4. Cloudflare: A record → <cloud provider IP>
5. Browser: TCP connect to <cloud provider IP>:443
6. Cloud provider serves the blog — homelab is not involved at all
```

The separation is structural: the Unifi wildcard only covers `homelab.davidsilva.dev`
and its subdomains. Anything directly under `davidsilva.dev` is invisible to it.

---

### What this configuration removes vs. the original compose setup

| Component | Original compose | This setup |
|---|---|---|
| AdGuard | Required (DNS + ad blocking) | Removed — Unifi handles both |
| Port-53 binding problem | Present | Gone — no DNS pod in Kubernetes |
| Static IPs for services | Required by AdGuard rewrites | Gone — Kubernetes DNS by service name |
| TLS certificates | Traefik certresolver (per-service) | cert-manager wildcard `*.homelab.davidsilva.dev` |
| Service routing | Docker socket labels | `IngressRoute` CRDs per service |
| Inter-service encryption | None (plain TCP on bridge network) | Istio mTLS (automatic) |
| `--docker-context` flag | Required | Replaced by kubeconfig context |
