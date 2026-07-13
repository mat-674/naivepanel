#!/bin/bash
# ====================================
# NaivePanel — One-Command Installer
# ====================================
#
# Usage:
#   bash <(curl -sL https://raw.githubusercontent.com/YOUR_USER/naivepanel/main/scripts/install.sh)
#
# Options:
#   --uninstall     Remove NaivePanel
#

set -euo pipefail

# --- Colors ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# --- Config ---
INSTALL_DIR="/opt/naivepanel"
SERVICE_NAME="naivepanel"
NAIVE_SERVICE_NAME="naiveproxy"
PANEL_REPO="mat-674/naivepanel"
CONFIG_FILE="${INSTALL_DIR}/config.json"

# --- Functions ---

print_banner() {
    echo -e "${CYAN}"
    echo "╔══════════════════════════════════════════╗"
    echo "║                                          ║"
    echo "║          NaivePanel Installer            ║"
    echo "║      NaiveProxy Management Panel         ║"
    echo "║                                          ║"
    echo "╚══════════════════════════════════════════╝"
    echo -e "${NC}"
}

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_ok() { echo -e "${GREEN}[OK]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root"
        exit 1
    fi
}

detect_os() {
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        OS=$ID
        OS_VERSION=$VERSION_ID
    elif [[ -f /etc/centos-release ]]; then
        OS="centos"
    else
        log_error "Unsupported OS"
        exit 1
    fi

    log_info "Detected OS: ${OS} ${OS_VERSION:-}"
}

detect_arch() {
    ARCH=$(uname -m)
    case $ARCH in
        x86_64)  ARCH="amd64" ;;
        aarch64) ARCH="arm64" ;;
        armv7l)  ARCH="arm" ;;
        *)
            log_error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac
    log_info "Architecture: $ARCH"
}

install_deps() {
    log_info "Installing dependencies..."
    case $OS in
        ubuntu|debian)
            apt-get update -qq
            apt-get install -y -qq curl tar jq file git golang-go build-essential > /dev/null 2>&1
            ;;
        centos|rhel|rocky|alma|fedora)
            yum install -y -q curl tar jq file git golang gcc > /dev/null 2>&1
            ;;
        arch|manjaro)
            pacman -Sy --noconfirm curl tar jq file git go base-devel > /dev/null 2>&1
            ;;
        *)
            log_warn "Unknown package manager, trying apt..."
            apt-get update -qq && apt-get install -y -qq curl tar jq file git golang-go build-essential > /dev/null 2>&1
            ;;
    esac

    # Check Go version
    if command -v go &> /dev/null; then
        GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
        if [[ $(echo -e "1.21\n$GO_VERSION" | sort -V | head -n1) != "1.21" ]]; then
            log_warn "Go version $GO_VERSION is too old. Need 1.21+. Attempting to install latest..."
        fi
    fi

    # Install xcaddy for building NaiveProxy
    if ! command -v xcaddy &> /dev/null; then
        log_info "Installing xcaddy..."
        if ! go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest > /dev/null 2>&1; then
            log_error "Failed to install xcaddy"
            return 1
        fi
        # Ensure xcaddy is in PATH
        export PATH=$PATH:$(go env GOPATH)/bin
        ln -sf "$(go env GOPATH)/bin/xcaddy" /usr/local/bin/xcaddy
    fi

    log_ok "Dependencies installed"
}

build_naive() {
    log_info "Building NaiveProxy (Caddy + forwardproxy) from source..."
    
    local BUILD_DIR="/tmp/naive_build"
    rm -rf "$BUILD_DIR"
    mkdir -p "$BUILD_DIR"
    cd "$BUILD_DIR"

    log_info "Compiling NaiveProxy (this will take a few minutes)..."
    # Build Caddy with forwardproxy plugin
    if ! xcaddy build v2.9.1 \
        --with github.com/caddyserver/forwardproxy=github.com/klzgrad/forwardproxy@master; then
        log_error "Failed to build NaiveProxy"
        cd - > /dev/null
        return 1
    fi

    mv caddy "${INSTALL_DIR}/naive"
    chmod +x "${INSTALL_DIR}/naive"
    cd - > /dev/null
    rm -rf "$BUILD_DIR"
    log_ok "NaiveProxy built and installed to ${INSTALL_DIR}/naive"
}

setup_wizard() {
    echo ""
    echo -e "${CYAN}╔══════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║          ${BOLD}Setup Wizard${NC}${CYAN}                    ║${NC}"
    echo -e "${CYAN}╚══════════════════════════════════════════╝${NC}"
    echo ""

    # --- Domain ---
    echo -e "${BOLD}1. Domain & TLS${NC}"
    echo -e "${YELLOW}   A domain is required for valid SSL. Make sure DNS A record points here.${NC}"
    echo -e "${YELLOW}   Leave empty to skip (self-signed TLS, may cause browser errors).${NC}"
    echo ""
    read -p "   Domain (e.g. proxy.example.com): " USER_DOMAIN
    if [[ -n "$USER_DOMAIN" ]]; then
        read -p "   Email for Let's Encrypt: " USER_TLS_EMAIL
        log_ok "Domain: ${USER_DOMAIN}"
    else
        USER_TLS_EMAIL=""
        log_warn "No domain. Self-signed TLS will be used."
    fi
    echo ""

    # --- Admin ---
    echo -e "${BOLD}2. Admin Account${NC}"
    echo -e "${YELLOW}   Used to log into the panel. Leave empty to auto-generate.${NC}"
    echo ""
    read -p "   Admin username [auto]: " USER_ADMIN_USER
    read -sp "   Admin password [auto]: " USER_ADMIN_PASS
    echo ""
    if [[ -n "$USER_ADMIN_USER" ]]; then
        log_ok "Admin: ${USER_ADMIN_USER}"
    else
        log_info "Admin credentials will be auto-generated."
    fi
    echo ""

    # --- Proxy User ---
    echo -e "${BOLD}3. First Proxy User${NC}"
    echo -e "${YELLOW}   NaiveProxy client credentials. Leave empty to auto-generate.${NC}"
    echo ""
    read -p "   Proxy username [auto]: " USER_PROXY_USER
    read -sp "   Proxy password [auto]: " USER_PROXY_PASS
    echo ""
    if [[ -n "$USER_PROXY_USER" ]]; then
        log_ok "Proxy user: ${USER_PROXY_USER}"
    else
        log_info "Proxy credentials will be auto-generated."
    fi
    echo ""

    echo -e "${GREEN}${BOLD}Setup wizard complete. Building...${NC}"
    echo ""
}

build_panel() {
    log_info "Building NaivePanel from source..."

    local BUILD_DIR="/tmp/naivepanel_build"
    rm -rf "$BUILD_DIR"

    log_info "Cloning repository: https://github.com/${PANEL_REPO}.git"
    if ! git clone "https://github.com/${PANEL_REPO}.git" "$BUILD_DIR" --quiet; then
        log_error "Failed to clone repository"
        return 1
    fi

    cd "$BUILD_DIR"
    log_info "Compiling (this may take a minute)..."
    if ! go build -ldflags="-s -w" -o "${INSTALL_DIR}/naivepanel" main.go; then
        log_error "Failed to build NaivePanel. Ensure Go is properly installed."
        cd - > /dev/null
        return 1
    fi

    chmod +x "${INSTALL_DIR}/naivepanel"
    mkdir -p "${INSTALL_DIR}/scripts"
    cp scripts/install.sh "${INSTALL_DIR}/scripts/install.sh"
    chmod +x "${INSTALL_DIR}/scripts/install.sh"
    cd - > /dev/null
    rm -rf "$BUILD_DIR"
    log_ok "NaivePanel built and installed to ${INSTALL_DIR}/naivepanel"

    # Setup database with wizard params
    log_info "Initializing database..."
    local SETUP_ARGS="--data-dir ${INSTALL_DIR} --setup --create-user"
    [[ -n "${USER_DOMAIN:-}" ]]      && SETUP_ARGS="${SETUP_ARGS} --domain ${USER_DOMAIN}"
    [[ -n "${USER_TLS_EMAIL:-}" ]]   && SETUP_ARGS="${SETUP_ARGS} --tls-email ${USER_TLS_EMAIL}"
    [[ -n "${USER_ADMIN_USER:-}" ]]  && SETUP_ARGS="${SETUP_ARGS} --admin-user ${USER_ADMIN_USER}"
    [[ -n "${USER_ADMIN_PASS:-}" ]]  && SETUP_ARGS="${SETUP_ARGS} --admin-pass ${USER_ADMIN_PASS}"
    [[ -n "${USER_PROXY_USER:-}" ]]  && SETUP_ARGS="${SETUP_ARGS} --proxy-user ${USER_PROXY_USER}"
    [[ -n "${USER_PROXY_PASS:-}" ]]  && SETUP_ARGS="${SETUP_ARGS} --proxy-pass ${USER_PROXY_PASS}"

    SETUP_OUTPUT=$("${INSTALL_DIR}/naivepanel" ${SETUP_ARGS} 2>&1 || true)

    # Parse output
    ADMIN_USER=$(echo "$SETUP_OUTPUT"  | grep "^Admin Username:" | awk -F': ' '{print $2}')
    ADMIN_PASS=$(echo "$SETUP_OUTPUT"  | grep "^Admin Password:" | awk -F': ' '{print $2}')
    PANEL_PORT=$(echo "$SETUP_OUTPUT"  | grep "^Panel Port:"     | awk -F': ' '{print $2}')
    PROXY_USER=$(echo "$SETUP_OUTPUT"  | grep "^Proxy Username:" | awk -F': ' '{print $2}')
    PROXY_PASS=$(echo "$SETUP_OUTPUT"  | grep "^Proxy Password:" | awk -F': ' '{print $2}')

    log_ok "Database initialized"
}

create_services() {
    log_info "Creating systemd services..."

    cat > /etc/systemd/system/${NAIVE_SERVICE_NAME}.service <<EOF
[Unit]
Description=NaiveProxy (Caddy + forwardproxy)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/naive run --config ${INSTALL_DIR}/Caddyfile --adapter caddyfile
ExecReload=${INSTALL_DIR}/naive reload --config ${INSTALL_DIR}/Caddyfile --adapter caddyfile
Restart=on-failure
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

    cat > /etc/systemd/system/${SERVICE_NAME}.service <<EOF
[Unit]
Description=NaivePanel - NaiveProxy Management Panel
After=network-online.target ${NAIVE_SERVICE_NAME}.service
Wants=network-online.target ${NAIVE_SERVICE_NAME}.service

[Service]
Type=simple
User=root
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/naivepanel --data-dir ${INSTALL_DIR} --bind 127.0.0.1
Restart=on-failure
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable ${NAIVE_SERVICE_NAME} > /dev/null 2>&1
    systemctl enable ${SERVICE_NAME} > /dev/null 2>&1
    log_ok "Systemd services created and enabled"
}

start_services() {
    log_info "Starting NaiveProxy and NaivePanel..."
    systemctl start ${NAIVE_SERVICE_NAME}
    systemctl start ${SERVICE_NAME}
    sleep 2

    if systemctl is-active --quiet ${NAIVE_SERVICE_NAME} && systemctl is-active --quiet ${SERVICE_NAME}; then
        log_ok "NaiveProxy and NaivePanel are running!"
    else
        log_error "Failed to start services. Check: journalctl -u ${NAIVE_SERVICE_NAME} and journalctl -u ${SERVICE_NAME}"
        return 1
    fi
}

show_credentials() {
    local IP=$(curl -s4 ifconfig.me 2>/dev/null || echo "YOUR_SERVER_IP")

    echo ""
    echo -e "${CYAN}╔═══════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║      ${BOLD}  NaivePanel — Installation Complete!  ${NC}${CYAN}       ║${NC}"
    echo -e "${CYAN}╠═══════════════════════════════════════════════════════╣${NC}"
    echo -e "${CYAN}║${NC}"
    echo -e "${CYAN}║${NC}  ${BOLD}📋 Admin Panel (loopback only)${NC}"
    echo -e "${CYAN}║${NC}     URL:       ${GREEN}http://127.0.0.1:${PANEL_PORT:-????}${NC}"
    echo -e "${CYAN}║${NC}     Tunnel:    ${GREEN}ssh -L ${PANEL_PORT:-????}:127.0.0.1:${PANEL_PORT:-????} root@${IP}${NC}"
    echo -e "${CYAN}║${NC}     Username:  ${GREEN}${ADMIN_USER:-N/A}${NC}"
    echo -e "${CYAN}║${NC}     Password:  ${GREEN}${ADMIN_PASS:-N/A}${NC}"
    echo -e "${CYAN}║${NC}"
    echo -e "${CYAN}║${NC}  ${BOLD}🔑 Proxy User${NC}"
    echo -e "${CYAN}║${NC}     Username:  ${GREEN}${PROXY_USER:-N/A}${NC}"
    echo -e "${CYAN}║${NC}     Password:  ${GREEN}${PROXY_PASS:-N/A}${NC}"

    if [[ -n "${USER_DOMAIN:-}" && -n "${PROXY_USER:-}" && -n "${PROXY_PASS:-}" ]]; then
        local URI="naive+https://${PROXY_USER}:${PROXY_PASS}@${USER_DOMAIN}:443"
        echo -e "${CYAN}║${NC}     URI:       ${GREEN}${URI}${NC}"
    fi

    echo -e "${CYAN}║${NC}"
    if [[ -n "${USER_DOMAIN:-}" ]]; then
        echo -e "${CYAN}║${NC}  ${BOLD}🌐 Domain:${NC}    ${GREEN}${USER_DOMAIN}${NC}"
        echo -e "${CYAN}║${NC}"
    fi
    echo -e "${CYAN}║${NC}  ${YELLOW}Panel logs:  journalctl -u ${SERVICE_NAME} -f${NC}"
    echo -e "${CYAN}║${NC}  ${YELLOW}Proxy logs:  journalctl -u ${NAIVE_SERVICE_NAME} -f${NC}"
    echo -e "${CYAN}║${NC}  ${RED}⚠  SAVE YOUR CREDENTIALS NOW!${NC}"
    echo -e "${CYAN}║${NC}"
    echo -e "${CYAN}╚═══════════════════════════════════════════════════════╝${NC}"
    echo ""
}

uninstall() {
    log_info "Uninstalling NaivePanel..."

    if systemctl is-active --quiet ${SERVICE_NAME} 2>/dev/null; then
        systemctl stop ${SERVICE_NAME}
        log_ok "Panel service stopped"
    fi

    if systemctl is-active --quiet ${NAIVE_SERVICE_NAME} 2>/dev/null; then
        systemctl stop ${NAIVE_SERVICE_NAME}
        log_ok "Proxy service stopped"
    fi

    if [[ -f /etc/systemd/system/${SERVICE_NAME}.service ]]; then
        systemctl disable ${SERVICE_NAME} > /dev/null 2>&1
        rm -f /etc/systemd/system/${SERVICE_NAME}.service
        systemctl daemon-reload
        log_ok "Panel service removed"
    fi

    if [[ -f /etc/systemd/system/${NAIVE_SERVICE_NAME}.service ]]; then
        systemctl disable ${NAIVE_SERVICE_NAME} > /dev/null 2>&1
        rm -f /etc/systemd/system/${NAIVE_SERVICE_NAME}.service
        systemctl daemon-reload
        log_ok "Proxy service removed"
    fi

    if [[ -d "$INSTALL_DIR" ]]; then
        read -p "Delete data directory ${INSTALL_DIR}? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            rm -rf "$INSTALL_DIR"
            log_ok "Data directory removed"
        else
            log_info "Data directory preserved"
        fi
    fi

    log_ok "NaivePanel uninstalled"
    exit 0
}

update_panel() {
    log_info "Updating NaivePanel..."

    if [[ ! -f "${INSTALL_DIR}/naivepanel" ]]; then
        log_error "NaivePanel is not installed at ${INSTALL_DIR}. Use the installer first."
        exit 1
    fi

    check_root
    detect_os
    detect_arch
    install_deps

    # Build the panel (this will clone latest and overwrite the binary)
    # We set dummy wizard vars so build_panel doesn't fail if it ever relies on them for setup
    USER_DOMAIN=""
    USER_TLS_EMAIL=""
    USER_ADMIN_USER=""
    USER_ADMIN_PASS=""
    USER_PROXY_USER=""
    USER_PROXY_PASS=""
    
    # Stop service before replacing binary
    if systemctl is-active --quiet ${SERVICE_NAME} 2>/dev/null; then
        log_info "Stopping service for update..."
        systemctl stop ${SERVICE_NAME}
    fi

    # build_panel clones to /tmp, builds, and moves to INSTALL_DIR
    # It also runs --setup, but we want to avoid resetting the DB if possible.
    # Actually build_panel calls setup_wizard if we are not careful.
    # Wait, build_panel in install.sh:205 doesn't call setup_wizard, it is called in main.
    
    log_info "Building latest NaivePanel..."
    local BUILD_DIR="/tmp/naivepanel_build"
    rm -rf "$BUILD_DIR"
    git clone "https://github.com/${PANEL_REPO}.git" "$BUILD_DIR" --quiet
    cd "$BUILD_DIR"
    go build -ldflags="-s -w" -o "${INSTALL_DIR}/naivepanel" main.go
    chmod +x "${INSTALL_DIR}/naivepanel"
    mkdir -p "${INSTALL_DIR}/scripts"
    cp scripts/install.sh "${INSTALL_DIR}/scripts/install.sh"
    chmod +x "${INSTALL_DIR}/scripts/install.sh"
    cd - > /dev/null
    rm -rf "$BUILD_DIR"

    log_ok "Binary updated"

    # Restart service
    create_services
    start_services
    log_ok "NaivePanel updated successfully!"
    exit 0
}

# --- Main ---

main() {
    print_banner

    # Check for flags
    if [[ "${1:-}" == "--uninstall" ]]; then
        check_root
        uninstall
    elif [[ "${1:-}" == "--update" ]]; then
        update_panel
    fi

    check_root
    detect_os
    detect_arch

    # Create install directory
    mkdir -p "$INSTALL_DIR"

    install_deps
    setup_wizard
    build_naive
    build_panel
    create_services
    start_services
    show_credentials
}

main "$@"
