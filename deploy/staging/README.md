# Staging (prepared, not deployed)

Doc 20 §3.2: one small domestic VM (4 vCore / 8 GB / 100 GB SSD), Docker Compose, TLS by Caddy. **Nothing here has been
provisioned or deployed.** The owner supplies the provider account and authorises the first deployment.

* `docker-compose.staging.yml` — overlay on `deploy/compose/docker-compose.yml`: `VIRA_ENV=staging`, no host ports except 80/443,
  Caddy reverse proxy with automatic HTTPS, devsim kept only for not-yet-contracted providers (allowed in staging, refused in production).
* `staging.env.example` — the variables the server's `.env` must define; real values live only on the server / secret store.
* `Caddyfile` — `api.`, `admin.` and `seller.` hostnames.

Deploy (after authorisation): copy the repository to the VM, create `.env` from `staging.env.example` with real secrets
(`chmod 600`), then:

```bash
docker compose --env-file .env -f deploy/compose/docker-compose.yml -f deploy/staging/docker-compose.staging.yml up -d --build --wait
```

Backups: nightly `pg_dump` to object storage (off-VM), restore drill monthly (doc 10 §8).
