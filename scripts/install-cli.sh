#!/bin/bash
# Rexec CLI Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/rexec/rexec/main/scripts/install-cli.sh | bash

set -e

REPO="rexec/rexec"
INSTALL_DIR="/usr/local/bin"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color
BOLD='\033[1m'

print_banner() {
    echo -e "${CYAN}${BOLD}"
    echo "██████╗ ███████╗██╗  ██╗███████╗ ██████╗"
    echo "██╔══██╗██╔════╝╚██╗██╔╝██╔════╝██╔════╝"
    echo "██████╔╝█████╗   ╚███╔╝ █████╗  ██║     "
    echo "██╔══██╗██╔══╝   ██╔██╗ ██╔══╝  ██║     "
    echo "██║  ██║███████╗██╔╝ ██╗███████╗╚██████╗"
    echo "╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚══════╝ ╚═════╝"
    echo -e "${NC}"
    echo -e "${BOLD}Cloud Terminal Environment CLI Installer${NC}"
    echo ""
}

detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case "$OS" in
        linux)
            OS="linux"
            ;;
        darwin)
            OS="darwin"
            ;;
        mingw*|msys*|cygwin*)
            OS="windows"
            ;;
        *)
            echo -e "${RED}Unsupported OS: $OS${NC}"
            exit 1
            ;;
    esac

    case "$ARCH" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        arm64|aarch64)
            ARCH="arm64"
            ;;
        *)
            echo -e "${RED}Unsupported architecture: $ARCH${NC}"
            exit 1
            ;;
    esac

    PLATFORM="${OS}-${ARCH}"
    echo -e "${GREEN}Detected platform: ${PLATFORM}${NC}"
}

get_latest_version() {
    echo -e "${CYAN}Fetching latest version...${NC}"
    VERSION=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    
    if [ -z "$VERSION" ]; then
        echo -e "${YELLOW}Could not fetch latest version, using v1.0.0${NC}"
        VERSION="v1.0.0"
    fi
    
    echo -e "${GREEN}Latest version: ${VERSION}${NC}"
}

download_binaries() {
    SUFFIX="${PLATFORM}"
    if [ "$OS" = "windows" ]; then
        SUFFIX="${SUFFIX}.exe"
    fi

    CLI_URL="https://github.com/${REPO}/releases/download/${VERSION}/rexec-cli-${SUFFIX}"
    TUI_URL="https://github.com/${REPO}/releases/download/${VERSION}/rexec-tui-${SUFFIX}"

    TEMP_DIR=$(mktemp -d)
    CLI_PATH="${TEMP_DIR}/rexec"
    TUI_PATH="${TEMP_DIR}/rexec-tui"

    if [ "$OS" = "windows" ]; then
        CLI_PATH="${CLI_PATH}.exe"
        TUI_PATH="${TUI_PATH}.exe"
    fi

    echo -e "${CYAN}Downloading rexec-cli...${NC}"
    if ! curl -fsSL "$CLI_URL" -o "$CLI_PATH"; then
        echo -e "${RED}Failed to download rexec-cli${NC}"
        exit 1
    fi

    echo -e "${CYAN}Downloading rexec-tui...${NC}"
    if ! curl -fsSL "$TUI_URL" -o "$TUI_PATH"; then
        echo -e "${YELLOW}Failed to download rexec-tui (optional)${NC}"
    fi

    chmod +x "$CLI_PATH" 2>/dev/null || true
    chmod +x "$TUI_PATH" 2>/dev/null || true

    echo "$TEMP_DIR"
}

install_binaries() {
    TEMP_DIR=$1

    echo -e "${CYAN}Installing to ${INSTALL_DIR}...${NC}"

    # Check if we need sudo
    if [ -w "$INSTALL_DIR" ]; then
        SUDO=""
    else
        SUDO="sudo"
        echo -e "${YELLOW}Requires sudo access to install to ${INSTALL_DIR}${NC}"
    fi

    if [ "$OS" = "windows" ]; then
        $SUDO mv "${TEMP_DIR}/rexec.exe" "${INSTALL_DIR}/rexec.exe"
        [ -f "${TEMP_DIR}/rexec-tui.exe" ] && $SUDO mv "${TEMP_DIR}/rexec-tui.exe" "${INSTALL_DIR}/rexec-tui.exe"
    else
        $SUDO mv "${TEMP_DIR}/rexec" "${INSTALL_DIR}/rexec"
        [ -f "${TEMP_DIR}/rexec-tui" ] && $SUDO mv "${TEMP_DIR}/rexec-tui" "${INSTALL_DIR}/rexec-tui"
    fi

    rm -rf "$TEMP_DIR"
}

verify_installation() {
    echo ""
    if command -v rexec &> /dev/null; then
        echo -e "${GREEN}${BOLD}✓ Installation successful!${NC}"
        echo ""
        rexec version
    else
        echo -e "${RED}Installation may have failed. Please check if ${INSTALL_DIR} is in your PATH.${NC}"
        exit 1
    fi
}

show_next_steps() {
    echo ""
    echo -e "${BOLD}Next steps:${NC}"
    echo ""
    echo -e "  1. Login to rexec:"
    echo -e "     ${CYAN}rexec login${NC}"
    echo ""
    echo -e "  2. List your terminals:"
    echo -e "     ${CYAN}rexec ls${NC}"
    echo ""
    echo -e "  3. Create a terminal:"
    echo -e "     ${CYAN}rexec create --name mydev${NC}"
    echo ""
    echo -e "  4. Connect to a terminal:"
    echo -e "     ${CYAN}rexec connect <terminal-id>${NC}"
    echo ""
    echo -e "  5. Launch interactive TUI:"
    echo -e "     ${CYAN}rexec -i${NC}"
    echo ""
    echo -e "${BOLD}Documentation:${NC} https://rexec.pipeops.io/docs"
    echo ""
}

main() {
    print_banner
    detect_platform
    get_latest_version
    TEMP_DIR=$(download_binaries)
    install_binaries "$TEMP_DIR"
    verify_installation
    show_next_steps
}

main "$@"
