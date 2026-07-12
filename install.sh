#!/bin/bash
set -e

# Mtop Universal Installer Script
# Detects distribution and sets up the APT repository or downloads the native binary

echo "=== Mtop System Monitor Installer ==="

# Check for root privilege
if [ "$EUID" -ne 0 ]; then
  echo "Please run as root (using sudo)."
  exit 1
fi

# Detect CPU Architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)
    ARCH_DEB="amd64"
    ARCH_BIN="amd64"
    ;;
  aarch64)
    ARCH_DEB="arm64"
    ARCH_BIN="arm64"
    ;;
  armv7l|armhf)
    ARCH_DEB="armhf"
    ARCH_BIN="arm"
    ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

# Detect OS Distribution
if [ -f /etc/os-release ]; then
  . /etc/os-release
  OS_ID=$ID
  OS_LIKE=$ID_LIKE
else
  OS_ID="unknown"
  OS_LIKE="unknown"
fi

# Function: Install on Debian/Ubuntu/Proxmox using APT Repository
install_apt() {
  echo "Debian/Ubuntu-based OS detected."
  echo "Adding Mtop APT repository..."

  # Create sources list entry with [trusted=yes] for easy zero-setup install
  echo "deb [trusted=yes] https://raw.githubusercontent.com/maui2023/mtop/main/apt stable main" > /etc/apt/sources.list.d/mtop.list

  echo "Updating package lists..."
  apt-get update -o Dir::Etc::sourcelist="sources.list.d/mtop.list" -o Dir::Etc::sourceparts="-" -o APT::Get::List-Cleanup="0"

  echo "Installing mtop..."
  apt-get install -y --allow-unauthenticated mtop
  echo "Mtop installed successfully! Type 'mtop' to run."
}

# Function: Install on Fedora/RHEL/CentOS/Rocky/Alma using DNF and precompiled binary
install_binary() {
  echo "Fedora/RedHat/DNF-based or other OS detected ($OS_ID)."
  echo "Downloading the precompiled binary for $ARCH..."

  URL="https://raw.githubusercontent.com/maui2023/mtop/main/mtop-linux-${ARCH_BIN}"

  echo "Downloading from: $URL"
  curl -L -o /usr/local/bin/mtop "$URL"
  chmod +x /usr/local/bin/mtop

  echo "Mtop installed successfully to /usr/local/bin/mtop! Type 'mtop' to run."
}

# Run installation based on OS
if [ "$OS_ID" = "debian" ] || [ "$OS_ID" = "ubuntu" ] || [[ "$OS_LIKE" == *"debian"* ]] || [[ "$OS_LIKE" == *"ubuntu"* ]]; then
  install_apt
else
  install_binary
fi
EOF
