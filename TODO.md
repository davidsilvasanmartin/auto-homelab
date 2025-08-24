# TODO

- REMOVE ALL HEALTH CHECKS when Uptime Kuma is completely set up
- ^^ UPDATE on previous point: NO !!! I believe health checks arae used by `depends_on` . Otherwise
  when we do `docker compose up` , many services crash because they can't connect to the services they depend on.
  If those services they depend on had health checks, all should work fine.
- [DONE] Pass credentials from env variables
    - It may be better to just create the credentials when the services start up
- [DONE] One big .env file with env vars for all services (properly namespaced)
- [DONE] Volumes should be configurable. These configs are for testing, real configs will be network volumes
- [DONE] Use the DNS server to give names to different services, e.g. adguard.home, calibre.home, and so on...
- Extra services
    - Probably having a few mdBook is better for my own documentation than Bookstack
    - Nextcloud ?? Will this be needed with TrueNAS ?? Good installing it if I want to keep the NAS "hidden"
    - Recipes
        - Tandoor Recipes: https://github.com/tandoorrecipes/recipes/
        - Mealie: https://github.com/mealie-recipes/mealie/
    - Stirling PDF: https://github.com/stirling-tools/stirling-pdf/
    - Jellyfin for video (alternatives: plex, emby)
    - My own project (yay!) for music, or one of the following:
        - Navidrome: https://github.com/navidrome/navidrome/
    - Monitoring: Mimir, Grafana
    - Logging: Loki ?? OpenSearch ??
    - Invidious (YouTube front-end): https://github.com/iv-org/invidious/
    - Changedetection.io for detecting changes on websites ?? https://github.com/dgtlmoon/changedetection.io/
    - Long-ish term project for home:
        - Home Assistant
    - Frigate, for cameras: https://github.com/blakeblackshear/frigate/
    - Homepage ??
    - Perhaps Ghostfolio (Angular!), or I can keep PortfolioPerformance. People say Ghostfolio has important accounting
      errors.
        - Alternative: maybe-finance: https://github.com/maybe-finance/maybe
    - Calibre-web-automated-book-downloader: https://github.com/calibrain/calibre-web-automated-book-downloader
    - Jelu is a self-hosted alternative to Goodreads: https://github.com/bayang/jelu
- Have a look at Reddit's survey to find more popular services: https://selfhosted-survey-2024.deployn.de/apps/
