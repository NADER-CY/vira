#!/usr/bin/env bash
# Creates .env (random local secrets) and the local JWT signing key pair. Safe to re-run: never overwrites.
set -euo pipefail
cd "$(dirname "$0")/.."
rand() { openssl rand -hex 24; }
if [ ! -f .env ]; then
  sed -e "s|^VIRA_PG_SUPERUSER_PASSWORD=.*|VIRA_PG_SUPERUSER_PASSWORD=$(rand)|" \
      -e "s|^MIGRATOR_PASSWORD=.*|MIGRATOR_PASSWORD=$(rand)|" \
      -e "s|^APP_PUBLIC_PASSWORD=.*|APP_PUBLIC_PASSWORD=$(rand)|" \
      -e "s|^APP_ADMIN_PASSWORD=.*|APP_ADMIN_PASSWORD=$(rand)|" \
      -e "s|^APP_WORKER_PASSWORD=.*|APP_WORKER_PASSWORD=$(rand)|" \
      -e "s|^VIRA_ADMIN_TOTP_KEY=.*|VIRA_ADMIN_TOTP_KEY=$(openssl rand -base64 32)|" \
      -e "s|^MINIO_ROOT_PASSWORD=.*|MINIO_ROOT_PASSWORD=$(rand)|" \
      -e "s|^VIRA_PII_ENC_KEY=.*|VIRA_PII_ENC_KEY=$(openssl rand -base64 32)|" \
      -e "s|^VIRA_PII_INDEX_KEY=.*|VIRA_PII_INDEX_KEY=$(openssl rand -base64 32)|" \
      .env.example > .env
  chmod 600 .env
  echo "created .env"
else
  # Add keys introduced after the file was created (never overwrites existing values).
  for k in VIRA_PII_ENC_KEY VIRA_PII_INDEX_KEY; do
    grep -q "^$k=" .env || echo "$k=$(openssl rand -base64 32)" >> .env
  done
  echo ".env exists — kept (missing keys added)"
fi
mkdir -p .data/jwt
if [ ! -f .data/jwt/private.pem ]; then
  openssl ecparam -name prime256v1 -genkey -noout | openssl pkcs8 -topk8 -nocrypt -out .data/jwt/private.pem
  openssl ec -in .data/jwt/private.pem -pubout -out .data/jwt/public.pem 2>/dev/null
  # Readable by the non-root container user. Development keys only.
  chmod 644 .data/jwt/private.pem .data/jwt/public.pem
  echo "created .data/jwt key pair (development only)"
fi
