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
            apt-get install -y -qq curl tar jq file git golang-go > /dev/null 2>&1
            ;;
        centos|rhel|rocky|alma|fedora)
            yum install -y -q curl tar jq file git golang > /dev/null 2>&1
            ;;
        arch|manjaro)
            pacman -Sy --noconfirm curl tar jq file git go > /dev/null 2>&1
            ;;
        *)
            log_warn "Unknown package manager, trying apt..."
            apt-get update -qq && apt-get install -y -qq curl tar jq file git golang-go > /dev/null 2>&1
            ;;
    esac
    log_ok "Dependencies installed"
}

download_naive() {
    log_info "Downloading NaiveProxy (caddy-forwardproxy)..."
    
    # Get latest release URL from klzgrad/naern
    local DOWNLOAD_URL
    
    # NaiveProxy releases binary names: naive-linux-{arch}
    # Try to get latest from GitHub API
    local RELEASE_URL="https://api.github.com/repos/klzgrad/naern/releases/latest"
    local RELEASE_INFO
    RELEASE_INFO=$(curl -sL "$RELEASE_URL" 2>/dev/null || echo "")
    
    if [[ -z "$RELEASE_INFO" || "$RELEASE_INFO" == *"Not Found"* ]]; then
        # Fallback: download from known working builds
        log_warn "Could not fetch latest release, using fallback..."
        DOWNLOAD_URL="https://github.com/nicennnnnnnlee/naern/releases/latest/download/naive-linux-${ARCH}"
    else
        DOWNLOAD_URL=$(echo "$RELEASE_INFO" | jq -r ".assets[] | select(.name | contains(\"naive-linux-${ARCH}\")) | .browser_download_url" | head -1)
    fi

    if [[ -z "$DOWNLOAD_URL" || "$DOWNLOAD_URL" == "null" ]]; then
        log_error "Could not find NaiveProxy binary for linux-${ARCH}"
        log_info "You can manually place the naive binary at: ${INSTALL_DIR}/naive"
        return 1
    fi

    curl -#L "$DOWNLOAD_URL" -o "${INSTALL_DIR}/naive"
    chmod +x "${INSTALL_DIR}/naive"
    log_ok "NaiveProxy downloaded to ${INSTALL_DIR}/naive"
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

    echo -e "${CYAN}║${NC}"
    echo -e "${CYAN}║${NC}  Check logs:  ${YELLOW}journalctl -u ${SERVICE_NAME} -f${NC}"
    echo -e "${CYAN}║${NC}  Credentials are shown in the service logs"
    echo -e "${CYAN}║${NC}  Run: ${YELLOW}journalctl -u ${SERVICE_NAME} | head -20${NC}"
    echo -e "${CYAN}║${NC}"
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
    download_naive
    build_panel
    create_service
    start_service
    show_credentials
}

main "$@"
