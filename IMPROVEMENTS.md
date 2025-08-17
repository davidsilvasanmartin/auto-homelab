# Taking This Home Server Project to the Next Level

This document outlines pragmatic improvements to make your setup easier to use, more robust, and aligned with popular, well‑supported solutions. It is tailored to the current repository, which orchestrates:

- Core network services: AdGuard Home + Unbound
- Reverse proxy: Nginx (templated)
- Apps: Calibre Web Automated, Paperless‑NGX (+ Postgres + Redis), Immich (+ ML + Valkey + Postgres), Linkwarden (+ Postgres), Jellyfin, Firefly‑III (+ MariaDB), Portainer, Uptime Kuma
- Backup scripts using Restic + Backblaze B2

The recommendations are grouped by priority and theme, with actionable steps and references.


## 1) Quick Wins (High Impact, Low Effort)

- Validate your compose file in CI and pre‑commit hooks
  - Add: `docker compose config` (syntax + env expansion check), `docker compose --ansi never config > /dev/null` for automation
  - Add pre-commit hook with `yamllint` and `dotenv-linter`
- Add `healthcheck:` to all stateful services and use `depends_on: { condition: service_healthy }`
  - Enables deterministic startup order and safer restarts
  - Example (Postgres):
    ```yaml
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U $POSTGRES_USER -d $POSTGRES_DB"]
      interval: 10s
      timeout: 5s
      retries: 10
    ```
- Stop hardcoding secrets in comments and ensure `.env.example` is complete
  - Remove example passwords from compose comments to prevent accidental reuse and leakage
  - Ensure `.env.example` has every variable used in compose with safe defaults and comments
- Pin image versions with digests where possible
  - Many services are pinned by tag; add digests for reproducibility (popular best practice):
    `image: nginx:1.29.0-bookworm@sha256:<digest>`
- Standardize restart policies
  - You already use `restart: unless-stopped`; apply uniformly and add healthchecks to avoid crash loops masking failures


## 2) Developer Experience & Ease of Use

- Introduce a Makefile (or Taskfile) with common workflows
  - Examples:
    - `make bootstrap` (copy .env.example -> .env, generate secrets, pull images)
    - `make up` / `make down` / `make restart` / `make logs svc=<name>`
    - `make validate` (compose + lint)
- Profiles for optional stacks
  - Use `profiles:` in services to toggle groups: `media`, `docs`, `photos`, `finance`, etc.
  - Example: `docker compose --profile media up -d`
- Home dashboard for discoverability
  - Add a landing page like Homepage or Homarr to list all services, links, health, and shortcuts
    - Popular: https://github.com/gethomepage/homepage or https://github.com/ajnart/homarr
- Declarative onboarding
  - Script to generate strong secrets into `.env` (e.g., using `openssl rand -base64 32`), reduce manual steps
  - Add comments in `.env.example` explaining each variable and safe defaults


## 3) Networking & Reverse Proxy

- Prefer service discovery over static IPs
  - Docker’s internal DNS resolves `service_name` automatically. Static IPs increase maintenance and risk of conflicts
  - Keep a custom network, but drop fixed `ipv4_address` unless you have compelling reasons
- Use DNS names for internal routing and local TLD
  - Leverage AdGuard/Unbound for local domains: `adguard.home`, `paperless.home`, `immich.home`, etc. (already on your TODO)
- Consider switching to a modern, popular reverse proxy with auto‑TLS and service discovery
  - Traefik (very popular): automatic Let’s Encrypt, routing by Docker labels, middleware (auth, headers, rate limits)
    - https://doc.traefik.io/traefik/
  - Alternatively, Caddy for super simple auto‑TLS and config
    - https://caddyserver.com/docs/
  - Advantages vs templated Nginx: less boilerplate, per‑service labels, dynamic config, easier multi-domain
- Secure headers and HTTPS by default
  - Whether you keep Nginx or move to Traefik/Caddy, enforce HTTPS, HSTS, X-Frame-Options, CSP, etc.
- Remote access options (zero‑trust)
  - Popular options: Cloudflare Tunnel, Tailscale Funnel, or Tailscale + ACLs for private access


## 4) Security Hardening

- Run as non‑root where possible
  - Many images support `user: "1000:1000"`; set explicit PUID/PGID (you already have for some) and drop capabilities
- Read‑only root filesystems and tmpfs
  - `read_only: true` and mount `tmpfs: [/tmp]` for services that do not write to root
- Capabilities and seccomp
  - Add `cap_drop: [ALL]` and selectively `cap_add` only as needed (DNS servers, etc.)
  - Add `security_opt: ["no-new-privileges:true"]`
- Network segmentation
  - Consider separate networks (e.g., `frontend`, `backend`, `db`) to limit east‑west traffic
- Secrets management
  - Move sensitive material out of `.env` where practical: Docker `secrets:` or SOPS‑encrypted `.env` checked into repo
  - Popular approach: `sops` + `age` for repo‑stored secrets; decrypt at deploy time


## 5) Reliability, Backups, and Disaster Recovery

- Make backups comprehensive and automated
  - You already use Restic + B2 via scripts; schedule with cron/systemd timers or a containerized scheduler (e.g., Ofelia)
  - Back up: app data volumes, uploads, AND database dumps (hot logical backups)
    - Postgres: `pg_dumpall` or per‑db `pg_dump` into a mounted backup dir, then Restic picks it up
    - MariaDB: `mysqldump --single-transaction --routines --triggers`
- Backup verification and retention
  - Add `restic check`, `restic forget --prune` with clear retention policy (e.g., 7 daily, 4 weekly, 12 monthly)
  - Periodic test restore to a staging path; document restore procedures per service
- Snapshots and consistency
  - For high‑write services (Immich DB), consider periodic `docker exec` dump + `fsync=on` and Postgres data checksums (already enabled)
  - For Redis/Valkey, decide on persistence (RDB/AOF) and back it up if needed
- Versioned configuration and bootstraps
  - Keep all critical templates under version control and include a bootstrap that can recreate missing directories/volumes


## 6) Observability and Operations

- Health checks everywhere
  - Add `healthcheck` to web apps, databases, Redis/Valkey; rely on `curl -f` or app‑specific commands
- Proactive updates and notifications
  - Add Diun (popular) to get image update notifications: https://crazymax.dev/diun/
  - If you want auto‑updates, Watchtower with caution and maintenance windows
- Centralized logs
  - Popular stack: Grafana Loki + Promtail (compose labels or promtail scrape configs)
  - Minimal alternative: Dozzle for live container logs UI
- Metrics and dashboards
  - cAdvisor + node‑exporter + Grafana + Prometheus (popular and well‑documented)
  - Uptime Kuma is already present for synthetic checks; add HTTP checks for each service via reverse proxy


## 7) Compose Best Practices and Structure

- Use anchors to DRY repetitive blocks
  - Define `x-logging`, `x-healthcheck`, `x-env`, `x-common-labels` and reuse
- Consistent environment variable namespacing
  - You already namespace (`PAPERLESS_`, `IMMICH_`, etc.). Ensure consistency and document them in `.env.example`
- Volumes and paths
  - Keep a uniform layout for data/config/cache per service; a central `data/` root improves portability
- Labels for routing and observability
  - If adopting Traefik, put all routing rules in Docker labels (domain, middleware, TLS, service)
- CI for drift detection
  - Add a GitHub Action to run `docker compose pull` and `docker compose config` to detect failures early


## 8) Authentication, Authorization, and SSO

- Perimeter auth for sensitive apps
  - Authelia (popular) or authentik for SSO in front of services via reverse proxy middleware
  - Enforce 2FA and strong policies for administrative apps (Portainer, Firefly, Jellyfin admin, etc.)


## 9) Reverse Proxy Migration Blueprint (Traefik Example)

- Add Traefik service:
  ```yaml
  traefik:
    image: traefik:v3.1
    command:
      - --providers.docker=true
      - --entrypoints.web.address=:80
      - --entrypoints.websecure.address=:443
      - --certificatesresolvers.letsencrypt.acme.httpchallenge=true
      - --certificatesresolvers.letsencrypt.acme.httpchallenge.entrypoint=web
      - --certificatesresolvers.letsencrypt.acme.email=${LETSENCRYPT_EMAIL}
      - --certificatesresolvers.letsencrypt.acme.storage=/letsencrypt/acme.json
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./traefik:/letsencrypt
    restart: unless-stopped
  ```
- Add labels to a service (example: Paperless):
  ```yaml
  labels:
    - "traefik.enable=true"
    - "traefik.http.routers.paperless.rule=Host(`paperless.${GLOBAL_DOMAIN}`)"
    - "traefik.http.routers.paperless.entrypoints=websecure"
    - "traefik.http.routers.paperless.tls.certresolver=letsencrypt"
    - "traefik.http.services.paperless.loadbalancer.server.port=8000" # match container port
  ```
- Remove static IPs, rely on service names on the shared network


## 10) Secrets and Configuration Management

- Move credentials out of compose comments
  - Replace with `docker secret` or env variables from `.env`, and generate them on first run
- Use SOPS for repository‑stored secrets
  - Encrypt `.env` values with `sops` + `age`; decrypt on the host during deploy
- Parameterize sensitive configs
  - Ensure `APP_KEY`, `NEXTAUTH_SECRET`, DB passwords, etc. are generated and rotated


## 11) Service‑Specific Notes

- AdGuard + Unbound
  - Consider redundancy (second Unbound instance) and smart upstreams; add healthchecks to avoid DNS outages
- Immich
  - Ensure DB backups include `pg_dump` with extension metadata; consider GPU acceleration profiles if applicable
  - Validate reverse proxy websockets and upload size limits (common cause of “Server offline” in frontend)
- Paperless‑NGX
  - Consider OCR cache sizing and persistent Redis config; backup `data/` and `media/` plus DB
- Linkwarden
  - Meilisearch is optional for search; document the trade‑offs and resource cost
- Jellyfin
  - If transcoding, configure hardware acceleration labels/devices and consider mounting fonts for subtitles
- Firefly‑III
  - Ensure APP_KEY is strong and persistent; schedule DB dumps
- Portainer
  - Restrict exposure to LAN or behind SSO; enable TLS if exposed


## 12) Popular Add‑Ons to Consider

- SSO: Authelia or authentik
- Dashboard: Homepage or Homarr
- Monitoring: Prometheus + Grafana
- Logs: Loki + Promtail; Dozzle for quick view
- Updates: Diun (notifications) or Watchtower (auto‑update cautiously)
- Remote access: Cloudflare Tunnel, Tailscale


## 13) Example Healthchecks

- Generic HTTP service:
  ```yaml
  healthcheck:
    test: ["CMD-SHELL", "curl -fsS http://localhost:PORT/health || exit 1"]
    interval: 15s
    timeout: 3s
    retries: 10
  ```
- Redis/Valkey:
  ```yaml
  healthcheck:
    test: ["CMD", "redis-cli", "ping"]
    interval: 10s
    timeout: 3s
    retries: 10
  ```


## 14) Example Makefile

```makefile
.PHONY: bootstrap up down restart logs validate pull

bootstrap:
	cp -n .env.example .env || true
	@echo "Generating secrets if missing..." # implement a small script here
	docker compose pull

up:
	docker compose up -d
	docker compose ps
	docker compose ls

down:
	docker compose down

restart:
	docker compose down; docker compose up -d

logs:
	docker compose logs -f $${svc}

validate:
	docker compose config > /dev/null
	yamllint docker-compose.yml
	dotenv-linter .env.example

pull:
	docker compose pull
```


## 15) CI Suggestions (GitHub Actions)

- On PR/Push:
  - `docker compose config` (validate)
  - `yamllint` and `dotenv-linter`
  - Optionally `trivy config` for IaC scan and `trivy image` on pinned digests for critical services


## 16) Documentation Hygiene

- Link this document from README
- Keep `.env.example` authoritative; include inline comments for each variable
- Add per‑service READMEs under `_documentation/<service>/README.md` for special steps (e.g., Immich reverse proxy notes, GPU acceleration)


---

By implementing the “Quick Wins” and adopting Traefik (or Caddy) with labels and healthchecks, you’ll significantly improve ease‑of‑use and reliability. Layer in automated backups with verified restores, observability (Loki/Promtail + Grafana/Prometheus), and a dashboard (Homepage/Homarr) to make day‑to‑day operations smooth and resilient. These are popular, well‑supported components with strong communities and documentation.