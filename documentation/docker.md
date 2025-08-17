# Restart Policy in Docker Compose

The `restart: unless-stopped` directive is a restart policy that tells Docker how to handle container restarts. Let me
explain this policy and answer your other questions:

## What does `restart: unless-stopped` do?

The `restart: unless-stopped` policy configures the container to:

- Automatically restart if it exits due to an error
- Automatically restart if the Docker daemon or the host system restarts
- **NOT** restart if the container was manually stopped by the user

This is different from other restart policies like:

- `no` - Never restart (default)
- `always` - Always restart regardless of exit status, even if manually stopped
- `on-failure[:max-retries]` - Only restart if the container exits with a non-zero status code

## Is this advisable for the adguard service?

Yes, adding `restart: unless-stopped` to the adguard service would be advisable because:

- It ensures AdGuard Home stays running even after system reboots
- It automatically recovers from crashes
- It won't fight against you if you deliberately stop the service
- For a home network DNS service like AdGuard, continuous availability is important
