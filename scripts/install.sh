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
NAIVE_REPO="klzgrad/naern"
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
    if ! xcaddy build \
        --with github.com/klzgrad/forwardproxy@master; then
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
    cd - > /dev/null
    rm -rf "$BUILD_DIR"
    log_ok "NaivePanel built and installed to ${INSTALL_DIR}/naivepanel"

    # Setup database and capture credentials
    log_info "Initializing database..."
    SETUP_OUTPUT=$("${INSTALL_DIR}/naivepanel" --data-dir "${INSTALL_DIR}" --setup || true)
    ADMIN_USER=$(echo "$SETUP_OUTPUT" | grep "Admin Username:" | awk '{print $4}')
    ADMIN_PASS=$(echo "$SETUP_OUTPUT" | grep "Admin Password:" | awk '{print $4}')
}

create_service() {
    log_info "Creating systemd service..."

    cat > /etc/systemd/system/${SERVICE_NAME}.service <<EOF
[Unit]
Description=NaivePanel - NaiveProxy Management Panel
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/naivepanel --data-dir ${INSTALL_DIR}
Restart=on-failure
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable ${SERVICE_NAME} > /dev/null 2>&1
    log_ok "Systemd service created and enabled"
}

start_service() {
    log_info "Starting NaivePanel..."
    systemctl start ${SERVICE_NAME}
    sleep 2

    if systemctl is-active --quiet ${SERVICE_NAME}; then
        log_ok "NaivePanel is running!"
    else
        log_error "Failed to start NaivePanel. Check: journalctl -u ${SERVICE_NAME}"
        return 1
    fi
}

show_credentials() {
    echo ""
    echo -e "${CYAN}╔══════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║        ${BOLD}NaivePanel Installed Successfully!${NC}${CYAN}        ║${NC}"
    echo -e "${CYAN}╠══════════════════════════════════════════════════╣${NC}"

    # Read config to show port
    if [[ -f "$CONFIG_FILE" ]]; then
        local PORT=$(jq -r '.panel_port' "$CONFIG_FILE")
        local IP=$(curl -s4 ifconfig.me 2>/dev/null || echo "YOUR_SERVER_IP")
        echo -e "${CYAN}║${NC}  Panel URL: ${GREEN}http://${IP}:${PORT}${NC}"
    fi

    if [[ -n "${ADMIN_USER:-}" && -n "${ADMIN_PASS:-}" ]]; then
        echo -e "${CYAN}║${NC}  Username:  ${GREEN}${ADMIN_USER}${NC}"
        echo -e "${CYAN}║${NC}  Password:  ${GREEN}${ADMIN_PASS}${NC}"
    fi

    echo -e "${CYAN}║${NC}"
    echo -e "${CYAN}║${NC}  Check logs:  ${YELLOW}journalctl -u ${SERVICE_NAME} -f${NC}"
    echo -e "${CYAN}║${NC}  ${RED}⚠  Save your admin credentials!${NC}"
    echo -e "${CYAN}╚══════════════════════════════════════════════════╝${NC}"
    echo ""
}

uninstall() {
    log_info "Uninstalling NaivePanel..."

    if systemctl is-active --quiet ${SERVICE_NAME} 2>/dev/null; then
        systemctl stop ${SERVICE_NAME}
        log_ok "Service stopped"
    fi

    if [[ -f /etc/systemd/system/${SERVICE_NAME}.service ]]; then
        systemctl disable ${SERVICE_NAME} > /dev/null 2>&1
        rm -f /etc/systemd/system/${SERVICE_NAME}.service
        systemctl daemon-reload
        log_ok "Service removed"
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

# --- Main ---

main() {
    print_banner

    # Check for uninstall flag
    if [[ "${1:-}" == "--uninstall" ]]; then
        check_root
        uninstall
    fi

    check_root
    detect_os
    detect_arch

    # Create install directory
    mkdir -p "$INSTALL_DIR"

    install_deps
    build_naive
    build_panel
    create_service
    start_service
    show_credentials
}

main "$@"
