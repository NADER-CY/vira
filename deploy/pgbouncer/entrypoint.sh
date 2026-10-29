#!/bin/sh
# Writes pgbouncer.ini and userlist.txt from environment variables (no secrets in the image or repository).
# Transaction pooling, as required by ADR-004 / doc 06: the application keeps all session state in SET LOCAL.
set -eu
dir=/tmp/pgbouncer
mkdir -p "$dir"
: > "$dir/userlist.txt"
for pair in "app_public:$APP_PUBLIC_PASSWORD" "app_admin:$APP_ADMIN_PASSWORD" "app_worker:$APP_WORKER_PASSWORD"; do
  user=${pair%%:*}; pass=${pair#*:}
  printf '"%s" "%s"\n' "$user" "$pass" >> "$dir/userlist.txt"
done
cat > "$dir/pgbouncer.ini" <<INI
[databases]
${DB_NAME} = host=${DB_HOST} port=${DB_PORT:-5432} dbname=${DB_NAME}

[pgbouncer]
listen_addr = 0.0.0.0
listen_port = 6432
auth_type = scram-sha-256
auth_file = $dir/userlist.txt
pool_mode = transaction
max_client_conn = 500
default_pool_size = 20
max_prepared_statements = 0
server_reset_query =
ignore_startup_parameters = extra_float_digits,options
admin_users =
stats_users =
INI
exec pgbouncer "$dir/pgbouncer.ini"
