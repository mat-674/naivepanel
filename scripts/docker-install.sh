#!/bin/bash
# ============================================================
# NaivePanel — Docker one-command installer
# Builds the image and brings the stack up with docker compose.
# ============================================================
set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'
log_info() { echo -e "${CYAN}[INFO]${NC} $1"; }
log_ok()   { echo -e "${GREEN}[OK]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error(){ echo -e "${RED}[ERROR]${NC} $1"; }

# Resolve repo root (this script lives in scripts/).
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

# --- Detect docker + compose ---
if ! command -v docker >/dev/null 2>&1; then
    log_error "docker is not installed. Install Docker Engine first: https://docs.docker.com/engine/install/"
    exit 1
fi
if docker compose version >/dev/null 2>&1; then
    COMPOSE="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
    COMPOSE="docker-compose"
else
    log_error "docker compose is not available. Install the Compose plugin: https://docs.docker.com/compose/install/"
    exit 1
fi
log_ok "Using: ${COMPOSE}"

# --- Host networking is Linux-only ---
if [[ "$(uname -s)" != "Linux" ]]; then
    log_warn "This stack uses host networking, which only works on Linux hosts."
    log_warn "On Docker Desktop (macOS/Windows) the loopback panel and public :443 will not bind to the host."
fi

# --- Handle --down / --uninstall ---
if [[ "${1:-}" == "--down" ]]; then
    log_info "Stopping the stack (data volume preserved)..."
    ${COMPOSE} down
    log_ok "Stopped. Data volume 'naivepanel-data' kept. Remove it with: docker volume rm naivepanel-data"
    exit 0
fi
if [[ "${1:-}" == "--uninstall" ]]; then
    log_warn "This removes the container AND the data volume (config, database, certificates)."
    read -rp "Type 'yes' to confirm: " CONFIRM
    if [[ "${CONFIRM}" == "yes" ]]; then
        ${COMPOSE} down -v
        log_ok "Stack and data volume removed."
    else
        log_info "Aborted."
    fi
    exit 0
fi

# --- Collect first-boot settings (only used on the very first run) ---
ENV_FILE="${REPO_ROOT}/.env"
if [[ -f "${ENV_FILE}" ]]; then
    log_info "Reusing existing .env for build-time settings."
else
    echo -e "${BOLD}Optional first-boot settings — press Enter to skip any field.${NC}"
    read -rp "  Domain (e.g. proxy.example.com): " IN_DOMAIN
    IN_TLS_EMAIL=""
    if [[ -n "${IN_DOMAIN}" ]]; then
        read -rp "  Email for Let's Encrypt: " IN_TLS_EMAIL
    fi
    {
        echo "DOMAIN=${IN_DOMAIN}"
        echo "TLS_EMAIL=${IN_TLS_EMAIL}"
    } > "${ENV_FILE}"
    log_ok "Wrote ${ENV_FILE}"
fi

# --- Build and start ---
log_info "Building image (compiles NaiveProxy + panel; first build takes a few minutes)..."
${COMPOSE} build

log_info "Starting the stack..."
${COMPOSE} up -d

# --- Wait for readiness and surface credentials ---
log_info "Waiting for first-boot initialisation..."
CREDS=""
for _ in $(seq 1 30); do
    LOGS="$(${COMPOSE} logs naivepanel 2>/dev/null || true)"
    if echo "${LOGS}" | grep -q "Admin Password:"; then
        CREDS="${LOGS}"
        break
    fi
    if echo "${LOGS}" | grep -q "NaivePanel starting on"; then
        # Already initialised on a previous run.
        CREDS="${LOGS}"
        break
    fi
    sleep 2
done

# Compose prefixes each log line with "naivepanel  | ", so match the label
# anywhere on the line and strip everything up to and including it.
extract() { echo "${CREDS}" | grep "$1" | head -n1 | sed "s/.*$1 *//" | tr -d '\r' || true; }

PANEL_PORT="$(echo "${CREDS}" | grep -oE "127\.0\.0\.1:[0-9]+" | head -n1 | cut -d: -f2 || true)"
[[ -z "${PANEL_PORT}" ]] && PANEL_PORT="$(extract 'Panel Port:')"
ADMIN_USER="$(extract 'Admin Username:')"
ADMIN_PASS="$(extract 'Admin Password:')"
PROXY_USER="$(extract 'Proxy Username:')"
PROXY_PASS="$(extract 'Proxy Password:')"

echo ""
echo -e "${CYAN}${BOLD}NaivePanel is running (Docker).${NC}"
if [[ -n "${ADMIN_PASS}" ]]; then
    echo -e "  ${BOLD}Admin:${NC} ${ADMIN_USER:-?} / ${ADMIN_PASS}"
    [[ -n "${PROXY_USER}" ]] && echo -e "  ${BOLD}Proxy user:${NC} ${PROXY_USER} / ${PROXY_PASS}"
    echo -e "  ${YELLOW}Save these now — they are only printed on first boot.${NC}"
else
    log_info "Already initialised previously. Retrieve logs with: ${COMPOSE} logs naivepanel"
fi
echo -e "  ${BOLD}Panel:${NC} http://127.0.0.1:${PANEL_PORT:-<port>} (loopback only)"
echo -e "  ${BOLD}SSH tunnel:${NC} ssh -L ${PANEL_PORT:-PORT}:127.0.0.1:${PANEL_PORT:-PORT} root@<server-ip>"
echo -e "  ${BOLD}Logs:${NC} ${COMPOSE} logs -f naivepanel"
echo ""

