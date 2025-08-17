# AdGuardHome

## Configuration notes
- IMPORTANT: the server you are running this homelab setup on must have AdGuard as its DNS resolver.
- WARNING: the IP of unbound is hardcoded in AdGuardHome.yaml. Unbound's port 5335 is also hardcoded.
- In AdGuardHome.yaml, the property `filtering: blocking_mode: refused` is required. If left default, it seems that
  AdGuardHome redirects blocked DNS requests to localhost. This causes Traefik to process them, convert them to HTTPS,
  and add its own self-signed certificate to them. This causes major problems.
