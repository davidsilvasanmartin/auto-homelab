# Kubernetes Migration: TLS Certificates

This document covers replacing Traefik's built-in `certresolver` (Cloudflare DNS challenge)
with `cert-manager`, the standard Kubernetes TLS certificate manager.

---

## Today's approach

Traefik handles TLS entirely. Service labels request a certificate:

```yaml
labels:
  - "traefik.http.routers.immich.tls.certresolver=cloudflare"
```

Traefik stores the issued certificate in the `HOMELAB_TRAEFIK_CERTS_PATH` directory. This
works, but it means:
- Certificates are managed as files on disk, not as Kubernetes objects
- No standard way to share a certificate across multiple services
- The cert lifecycle (renewal, rotation) is opaque to the rest of the cluster

---

## cert-manager

`cert-manager` is the de-facto standard for certificate management in Kubernetes. It issues
certificates as `Certificate` objects, stores them as `Secret` objects, and renews them
automatically. Traefik (and any other ingress controller) reads the `Secret`.

### Install cert-manager

```bash
helm repo add jetstack https://charts.jetstack.io
helm repo update

helm install cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --set crds.enabled=true
```

Verify:

```bash
kubectl get pods -n cert-manager
# cert-manager, cert-manager-cainjector, cert-manager-webhook — all Running
```

---

## Create a ClusterIssuer for Cloudflare DNS-01

The current setup uses Cloudflare's DNS API to prove domain ownership (DNS-01 challenge).
cert-manager supports this natively via the ACME provider.

First, store the Cloudflare API token as a Secret:

```bash
kubectl create secret generic cloudflare-api-token \
  --namespace cert-manager \
  --from-literal=api-token=<your_CF_DNS_API_TOKEN>
```

Then create the `ClusterIssuer`:

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-cloudflare
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: d.silvas@outlook.com
    privateKeySecretRef:
      name: letsencrypt-account-key
    solvers:
      - dns01:
          cloudflare:
            apiTokenSecretRef:
              name: cloudflare-api-token
              key: api-token
```

A `ClusterIssuer` (as opposed to a namespace-scoped `Issuer`) can issue certificates for
any namespace in the cluster. This is what you want for a homelab where all services share
the same domain.

---

## Wildcard certificate (recommended)

Rather than issuing one certificate per service, issue a single wildcard certificate for
`*.home.example.com`. This is simpler to manage and faster to issue.

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: homelab-wildcard
  namespace: homelab-system
spec:
  secretName: homelab-wildcard-tls   # Secret that Traefik will reference
  issuerRef:
    name: letsencrypt-cloudflare
    kind: ClusterIssuer
  commonName: "*.home.example.com"
  dnsNames:
    - "*.home.example.com"
    - "home.example.com"
  renewBefore: 360h   # renew 15 days before expiry
```

cert-manager will create (and keep up to date) a `Secret` named `homelab-wildcard-tls` in
the `homelab-system` namespace containing the TLS certificate and key.

---

## Configure Traefik to use the wildcard certificate

In Traefik's `IngressRoute`, reference the secret directly instead of using a certresolver:

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
    secretName: homelab-wildcard-tls
```

Or configure Traefik's default TLS store in its Helm values so all routes use the wildcard
without having to specify it per-route:

```yaml
# traefik helm values
tlsStore:
  default:
    defaultCertificate:
      secretName: homelab-wildcard-tls
```

With this, every `IngressRoute` with `tls: {}` (or no tls block) uses the wildcard cert
automatically — equivalent to the `certresolver=cloudflare` in every compose label today,
but without Traefik having to perform the ACME challenge itself.

---

## Cross-namespace certificate access

The wildcard Secret is in `homelab-system`. Ingress routes in the `homelab` namespace need
to read it. Two options:

**Option A**: Keep all `IngressRoute` objects in `homelab-system`. This concentrates all
TLS config in one namespace but couples routing to the system namespace.

**Option B**: Use Traefik's `TLSStore` (shown above) — Traefik reads the secret once and
applies it globally. Routes in any namespace get the wildcard cert transparently.

**Option C**: Use the `reflector` operator to sync Secrets across namespaces automatically.

For a homelab, Option B (global TLSStore default) is the simplest.

---

## Staging issuer for testing

Add a staging ClusterIssuer for testing certificate issuance before going to production.
Let's Encrypt has strict rate limits on the production endpoint.

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-staging
spec:
  acme:
    server: https://acme-staging-v02.api.letsencrypt.org/directory
    email: d.silvas@outlook.com
    privateKeySecretRef:
      name: letsencrypt-staging-key
    solvers:
      - dns01:
          cloudflare:
            apiTokenSecretRef:
              name: cloudflare-api-token
              key: api-token
```

Test by setting `issuerRef.name: letsencrypt-staging` in the `Certificate` object. The
staging cert is not trusted by browsers but proves the DNS challenge works. Switch to
`letsencrypt-cloudflare` (production) once confirmed.

---

## What improves over the compose approach

| Aspect | Today (Traefik certresolver) | With cert-manager |
|---|---|---|
| Certificate visibility | Opaque files on disk | `Certificate` and `Secret` objects, inspectable via `kubectl` |
| Renewal | Traefik handles it silently | cert-manager sends events, has Prometheus metrics |
| Sharing across services | One cert per service label | Single wildcard shared everywhere |
| Rotation without downtime | Depends on Traefik internals | cert-manager rotates the Secret; Traefik picks it up |
| Multiple domains/clusters | Not applicable | ClusterIssuer works across any namespace |

---

## Checking certificate status

```bash
kubectl describe certificate homelab-wildcard -n homelab-system
# Events will show: Issuing, Issued, Renewed, etc.

kubectl describe certificaterequest -n homelab-system
# Shows ACME challenge progress

kubectl get secret homelab-wildcard-tls -n homelab-system -o jsonpath='{.data.tls\.crt}' \
  | base64 -d | openssl x509 -noout -dates
# Shows actual expiry dates
```
