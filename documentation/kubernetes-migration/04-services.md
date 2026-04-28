# Kubernetes Migration: Service-by-Service Manifest Breakdown

This document translates each Docker Compose service into Kubernetes manifests. Only the
structural shape is shown — secrets are referenced by name and defined separately (see
`05-tls.md` for TLS, and the ConfigMap/Secret section below for credentials).

---

## Secrets and ConfigMaps strategy

### Today

Everything is in `.env`. Every container sees all variables.

### In Kubernetes

Create one `Secret` per service group (or per credential concern), and one `ConfigMap` for
non-sensitive configuration:

```bash
# Example: create paperless DB credentials as a Secret
kubectl create secret generic paperless-db-secret \
  --namespace homelab \
  --from-literal=POSTGRES_DB=paperless \
  --from-literal=POSTGRES_USER=paperless \
  --from-literal=POSTGRES_PASSWORD=<generated>

# Example: non-sensitive config as ConfigMap
kubectl create configmap paperless-config \
  --namespace homelab \
  --from-literal=PAPERLESS_URL=https://paperless.home.example.com
```

Secrets are base64-encoded in etcd. For stronger at-rest encryption, configure etcd
encryption or use Sealed Secrets / External Secrets Operator later.

---

## AdGuard

AdGuard requires a custom Docker image (with `apache2-utils`) and a startup script that
runs `htpasswd`. In Kubernetes this becomes an `initContainer`.

```yaml
# adguard-scripts configmap (content from files/adguard/scripts/)
apiVersion: v1
kind: ConfigMap
metadata:
  name: adguard-scripts
  namespace: homelab-system
binaryData:
  start-adguard.sh: |
    #!/bin/sh
    # ... content of files/adguard/scripts/start-adguard.sh
  AdGuardHome.yaml.template: |
    # ... content of the template
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: adguard
  namespace: homelab-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: adguard
  template:
    metadata:
      labels:
        app: adguard
    spec:
      hostNetwork: true        # required for port 53 — see 02-networking.md
      dnsPolicy: ClusterFirstWithHostNet
      initContainers:
        - name: configure
          image: adguard/adguardhome:with-apache2-utils
          command: ["/bin/sh", "/scripts/start-adguard.sh", "--configure-only"]
          env:
            - name: HOMELAB_GENERAL_DOMAIN
              valueFrom:
                configMapKeyRef:
                  name: homelab-general
                  key: DOMAIN
            - name: HOMELAB_ADGUARD_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: adguard-secret
                  key: PASSWORD
          volumeMounts:
            - name: conf
              mountPath: /opt/adguardhome/conf
            - name: scripts
              mountPath: /scripts
              readOnly: true
      containers:
        - name: adguard
          image: adguard/adguardhome:with-apache2-utils
          ports:
            - containerPort: 53
              protocol: UDP
            - containerPort: 53
              protocol: TCP
            - containerPort: 3000
          volumeMounts:
            - name: conf
              mountPath: /opt/adguardhome/conf
            - name: work
              mountPath: /opt/adguardhome/work
          livenessProbe:
            httpGet:
              path: /
              port: 3000
            initialDelaySeconds: 10
            periodSeconds: 30
      volumes:
        - name: conf
          persistentVolumeClaim:
            claimName: adguard-conf
        - name: work
          persistentVolumeClaim:
            claimName: adguard-work
        - name: scripts
          configMap:
            name: adguard-scripts
            defaultMode: 0755
---
# Web UI accessible via Traefik
apiVersion: v1
kind: Service
metadata:
  name: adguard-web
  namespace: homelab-system
spec:
  selector:
    app: adguard
  ports:
    - port: 3000
      targetPort: 3000
  type: ClusterIP
```

> **Note**: The startup script currently runs in the container entrypoint. It will need to
> be split: the configuration/password-writing phase goes into the `initContainer`, and
> AdGuard itself starts normally afterwards. Review `files/adguard/scripts/start-adguard.sh`
> when implementing this.

---

## Traefik

Traefik is installed via Helm (see `02-networking.md`). Per-service routing is configured
with `IngressRoute` CRDs or standard `Ingress` objects.

The Cloudflare DNS API credentials for TLS challenge become a `Secret`:

```bash
kubectl create secret generic cloudflare-credentials \
  --namespace homelab-system \
  --from-literal=CF_DNS_API_TOKEN=<token> \
  --from-literal=CF_DNS_API_EMAIL=<email>
```

The `traefik.yaml` static config is stored in a `ConfigMap` and mounted into the Traefik pod
via the Helm values:

```yaml
# values.yaml for helm install
volumes:
  - name: traefik-config
    configMap:
      name: traefik-static-config
additionalArguments:
  - "--configFile=/config/traefik.yaml"
```

---

## Calibre (calibre-web-automated)

Calibre is stateless at the process level (state lives in PVCs):

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: calibre
  namespace: homelab
spec:
  replicas: 1
  selector:
    matchLabels:
      app: calibre
  template:
    metadata:
      labels:
        app: calibre
    spec:
      securityContext:
        runAsUser: 1000
        runAsGroup: 1000
        fsGroup: 1000
      containers:
        - name: calibre
          image: crocodilestick/calibre-web-automated:V3.0.4
          env:
            - name: PUID
              value: "1000"
            - name: PGID
              value: "1000"
            - name: TZ
              value: Europe/London
          volumeMounts:
            - name: conf
              mountPath: /config
            - name: ingest
              mountPath: /cwa-book-ingest
            - name: library
              mountPath: /calibre-library
          ports:
            - containerPort: 8083
          readinessProbe:
            httpGet:
              path: /
              port: 8083
            initialDelaySeconds: 15
            periodSeconds: 10
      volumes:
        - name: conf
          persistentVolumeClaim:
            claimName: calibre-conf
        - name: ingest
          persistentVolumeClaim:
            claimName: calibre-ingest
        - name: library
          persistentVolumeClaim:
            claimName: calibre-library
---
apiVersion: v1
kind: Service
metadata:
  name: calibre
  namespace: homelab
spec:
  selector:
    app: calibre
  ports:
    - port: 8083
      targetPort: 8083
```

---

## Paperless-ngx

Paperless has three components: Redis, PostgreSQL, and the web application. The web
application must wait for both before starting — this is handled with `initContainers`.

### Redis

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: paperless-redis
  namespace: homelab
spec:
  serviceName: paperless-redis
  replicas: 1
  selector:
    matchLabels:
      app: paperless-redis
  template:
    metadata:
      labels:
        app: paperless-redis
    spec:
      containers:
        - name: redis
          image: redis:8.0.1-bookworm
          ports:
            - containerPort: 6379
          volumeMounts:
            - name: data
              mountPath: /data
  volumeClaimTemplates:
    - metadata:
        name: data
      spec:
        accessModes: [ReadWriteOnce]
        storageClassName: local-path
        resources:
          requests:
            storage: 500Mi
---
apiVersion: v1
kind: Service
metadata:
  name: paperless-redis
  namespace: homelab
spec:
  selector:
    app: paperless-redis
  ports:
    - port: 6379
```

### PostgreSQL (same StatefulSet pattern as shown in 03-storage.md)

Service name: `paperless-db`, port: `5432`.

### Paperless web

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: paperless
  namespace: homelab
spec:
  replicas: 1
  selector:
    matchLabels:
      app: paperless
  template:
    metadata:
      labels:
        app: paperless
    spec:
      initContainers:
        - name: wait-for-db
          image: postgres:17.5-bookworm
          command: ["sh", "-c",
            "until pg_isready -h paperless-db -p 5432; do sleep 2; done"]
        - name: wait-for-redis
          image: redis:8.0.1-bookworm
          command: ["sh", "-c",
            "until redis-cli -h paperless-redis ping; do sleep 2; done"]
      containers:
        - name: paperless
          image: paperlessngx/paperless-ngx:2.16.1
          env:
            - name: PAPERLESS_REDIS
              value: redis://paperless-redis:6379
            - name: PAPERLESS_DBHOST
              value: paperless-db
            - name: PAPERLESS_DBNAME
              valueFrom:
                secretKeyRef:
                  name: paperless-db-secret
                  key: POSTGRES_DB
            - name: PAPERLESS_DBUSER
              valueFrom:
                secretKeyRef:
                  name: paperless-db-secret
                  key: POSTGRES_USER
            - name: PAPERLESS_DBPASS
              valueFrom:
                secretKeyRef:
                  name: paperless-db-secret
                  key: POSTGRES_PASSWORD
            - name: PAPERLESS_URL
              value: https://paperless.home.example.com
          ports:
            - containerPort: 8000
          volumeMounts:
            - name: data
              mountPath: /usr/src/paperless/data
            - name: media
              mountPath: /usr/src/paperless/media
            - name: export
              mountPath: /usr/src/paperless/export
            - name: consume
              mountPath: /usr/src/paperless/consume
      volumes:
        - name: data
          persistentVolumeClaim:
            claimName: paperless-data
        - name: media
          persistentVolumeClaim:
            claimName: paperless-media
        - name: export
          persistentVolumeClaim:
            claimName: paperless-export
        - name: consume
          persistentVolumeClaim:
            claimName: paperless-consume
```

---

## Immich

Immich has four components: server, machine-learning, Redis (Valkey), and PostgreSQL.

The manifest structure mirrors Paperless: `StatefulSet` for databases, `Deployment` for
the application tier, `initContainers` for dependency ordering.

Key differences from the compose setup:
- `DB_HOSTNAME` changes from `${HOMELAB_IMMICH_DB_IP}` to `immich-db` (service name)
- `REDIS_HOSTNAME` changes to `immich-redis`
- The `UPLOAD_LOCATION` and `DB_DATA_LOCATION` env vars referenced in the Immich image
  are advisory; actual storage is via PVC mounts

```yaml
# Only showing the server deployment for brevity
apiVersion: apps/v1
kind: Deployment
metadata:
  name: immich
  namespace: homelab
spec:
  replicas: 1
  selector:
    matchLabels:
      app: immich
  template:
    metadata:
      labels:
        app: immich
    spec:
      initContainers:
        - name: wait-for-db
          image: postgres:14
          command: ["sh", "-c",
            "until pg_isready -h immich-db -p 5432; do sleep 2; done"]
        - name: wait-for-redis
          image: docker.io/valkey/valkey:8
          command: ["sh", "-c",
            "until redis-cli -h immich-redis ping; do sleep 2; done"]
        - name: wait-for-ml
          image: ghcr.io/immich-app/immich-machine-learning:v2.3.1
          command: ["sh", "-c",
            "until wget -qO- http://immich-machine-learning:3003/ping; do sleep 2; done"]
      containers:
        - name: immich
          image: ghcr.io/immich-app/immich-server:v2.3.1
          env:
            - name: DB_HOSTNAME
              value: immich-db
            - name: DB_DATABASE_NAME
              valueFrom:
                secretKeyRef:
                  name: immich-db-secret
                  key: POSTGRES_DB
            - name: DB_USERNAME
              valueFrom:
                secretKeyRef:
                  name: immich-db-secret
                  key: POSTGRES_USER
            - name: DB_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: immich-db-secret
                  key: POSTGRES_PASSWORD
            - name: REDIS_HOSTNAME
              value: immich-redis
            - name: TZ
              valueFrom:
                configMapKeyRef:
                  name: homelab-general
                  key: TIMEZONE
          ports:
            - containerPort: 2283
          volumeMounts:
            - name: uploads
              mountPath: /data
      volumes:
        - name: uploads
          persistentVolumeClaim:
            claimName: immich-uploads
```

---

## Portainer

Portainer manages Docker — but in a Kubernetes cluster it would manage Kubernetes instead.
The Portainer CE image supports Kubernetes and connects via the in-cluster API:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: portainer
  namespace: homelab-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: portainer
  template:
    metadata:
      labels:
        app: portainer
    spec:
      serviceAccountName: portainer   # needs RBAC to list/manage resources
      containers:
        - name: portainer
          image: portainer/portainer-ce:2.32.0
          args: ["-H", "unix:///var/run/docker.sock"]
          # In K8s mode, remove the docker.sock arg and use the K8s service account instead
          ports:
            - containerPort: 9000
          volumeMounts:
            - name: data
              mountPath: /data
      volumes:
        - name: data
          persistentVolumeClaim:
            claimName: portainer-data
```

> **Note**: In a Kubernetes cluster the more natural alternative to Portainer is `k9s` (a
> terminal UI) or the Kubernetes Dashboard. Portainer does support Kubernetes but it is less
> commonly used in that role.

---

## Firefly III

Firefly uses MariaDB. The MariaDB `StatefulSet` uses the same pattern as PostgreSQL above.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: firefly
  namespace: homelab
spec:
  replicas: 1
  selector:
    matchLabels:
      app: firefly
  template:
    spec:
      initContainers:
        - name: wait-for-db
          image: mariadb:11.8.2-noble
          command: ["sh", "-c",
            "until mariadb-admin ping -h firefly-db --silent; do sleep 2; done"]
      containers:
        - name: firefly
          image: fireflyiii/core:version-6.2.21
          env:
            - name: DB_HOST
              value: firefly-db
            - name: DB_PORT
              value: "3306"
            - name: APP_URL
              value: https://firefly.home.example.com
            # ... remaining vars from secretKeyRef / configMapKeyRef
          ports:
            - containerPort: 8080
          volumeMounts:
            - name: uploads
              mountPath: /var/www/html/storage/upload
      volumes:
        - name: uploads
          persistentVolumeClaim:
            claimName: firefly-uploads
```

---

## Navidrome

Navidrome is the simplest service — stateless process, two PVCs (data + music library):

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: navidrome
  namespace: homelab
spec:
  replicas: 1
  selector:
    matchLabels:
      app: navidrome
  template:
    spec:
      securityContext:
        runAsUser: 1000   # replaces compose `user: ${UID}:${GID}`
        runAsGroup: 1000
        fsGroup: 1000
      containers:
        - name: navidrome
          image: deluan/navidrome:0.58.5
          env:
            - name: ND_LOGLEVEL
              value: debug
          ports:
            - containerPort: 4533
          readinessProbe:
            httpGet:
              path: /ping
              port: 4533
            initialDelaySeconds: 5
            periodSeconds: 10
          volumeMounts:
            - name: data
              mountPath: /data
            - name: music
              mountPath: /music
              readOnly: true
      volumes:
        - name: data
          persistentVolumeClaim:
            claimName: navidrome-data
        - name: music
          persistentVolumeClaim:
            claimName: navidrome-music
```

---

## Recommended directory structure for manifests

```
k8s/
├── system/
│   ├── adguard/
│   │   ├── deployment.yaml
│   │   ├── services.yaml
│   │   ├── pvc.yaml
│   │   └── configmap-scripts.yaml
│   └── traefik/
│       ├── helm-values.yaml
│       └── configmap-static.yaml
├── homelab/
│   ├── calibre/
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   ├── pvc.yaml
│   │   └── ingress.yaml
│   ├── paperless/
│   │   ├── redis-statefulset.yaml
│   │   ├── db-statefulset.yaml
│   │   ├── web-deployment.yaml
│   │   ├── services.yaml
│   │   ├── pvc.yaml
│   │   ├── secret.yaml
│   │   └── ingress.yaml
│   ├── immich/
│   ├── firefly/
│   └── navidrome/
└── namespaces.yaml
```

This is the structural improvement that directly addresses the docker-compose.yml sprawl
problem. Each service is self-contained in its own directory.
