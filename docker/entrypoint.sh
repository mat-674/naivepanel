#!/bin/sh
# NaivePanel container entrypoint.
#
# Binaries are baked into the image; all mutable state (config.json, the
# SQLite database, the generated Caddyfile and Caddy's ACME certificates)
# lives in the /opt/naivepanel volume so it survives image upgrades.
set -eu

INSTALL_DIR="${INSTALL_DIR:-/opt/naivepanel}"
DB_FILE="${INSTALL_DIR}/naivepanel.db"

mkdir -p "${INSTALL_DIR}" "${INSTALL_DIR}/caddy-data"

# Refresh the naive binary into the volume on every boot so `docker compose
# pull && up` actually rolls out a new proxy build. The config's default
# NaiveBinary path points here.
cp -f /usr/local/bin/naive "${INSTALL_DIR}/naive"
chmod +x "${INSTALL_DIR}/naive"

# First boot: initialise the database and print credentials to the log. Keyed
# on the database file so restarts never regenerate the admin or proxy user.
if [ ! -f "${DB_FILE}" ]; then
    echo "[entrypoint] First boot: initialising database..."
    set -- --data-dir "${INSTALL_DIR}" --setup --create-user
    if [ -n "${DOMAIN:-}" ]; then     set -- "$@" --domain "${DOMAIN}"; fi
    if [ -n "${TLS_EMAIL:-}" ]; then  set -- "$@" --tls-email "${TLS_EMAIL}"; fi
    if [ -n "${ADMIN_USER:-}" ]; then set -- "$@" --admin-user "${ADMIN_USER}"; fi
    if [ -n "${ADMIN_PASS:-}" ]; then set -- "$@" --admin-pass "${ADMIN_PASS}"; fi
    if [ -n "${PROXY_USER:-}" ]; then set -- "$@" --proxy-user "${PROXY_USER}"; fi
    if [ -n "${PROXY_PASS:-}" ]; then set -- "$@" --proxy-pass "${PROXY_PASS}"; fi
    /usr/local/bin/naivepanel "$@"
    echo "[entrypoint] Initialisation complete."
fi

# Run the panel in the foreground (PID 1) so it receives SIGTERM on `docker
# stop` and shuts NaiveProxy down cleanly. --start-proxy makes the panel
# supervise NaiveProxy itself, since there is no systemd inside the container.
exec /usr/local/bin/naivepanel \
    --data-dir "${INSTALL_DIR}" \
    --bind 127.0.0.1 \
    --start-proxy
