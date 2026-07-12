#!/bin/bash
set -e

VERSION="1.0.0"
DEB_DIR="apt"
TMP_BUILD="tmp_build"

echo "=== Building mtop DEB Packages ==="

# Clean old directories
rm -rf "$DEB_DIR" "$TMP_BUILD"
mkdir -p "$DEB_DIR/pool/main/m/mtop"

# Build binaries first
echo "Compiling binaries..."
GOOS=linux GOARCH=amd64 go build -o mtop-amd64 main.go
GOOS=linux GOARCH=arm64 go build -o mtop-arm64 main.go
GOOS=linux GOARCH=arm GOARM=7 go build -o mtop-armhf main.go

# Helper function to create deb
create_deb() {
    local arch=$1
    local bin_suffix=$2
    local deb_arch=$3
    local build_dir="$TMP_BUILD/mtop_${VERSION}_${deb_arch}"

    echo "Packaging for $deb_arch..."
    mkdir -p "$build_dir/usr/bin"
    mkdir -p "$build_dir/DEBIAN"

    # Copy binary
    cp "mtop-$bin_suffix" "$build_dir/usr/bin/mtop"
    chmod 755 "$build_dir/usr/bin/mtop"

    # Create control file
    cat << EOF > "$build_dir/DEBIAN/control"
Package: mtop
Version: ${VERSION}
Section: utils
Priority: optional
Architecture: ${deb_arch}
Maintainer: maui2023 <https://github.com/maui2023>
Description: Premium System Monitor TUI Tailored for Proxmox LXC & Bare Metal
 Mtop is a lightweight terminal user interface (TUI) system monitoring application.
 It is highly optimized for Proxmox LXC containers and Bare Metal systems, providing
 accurate resource reporting by utilizing cgroups v2.
EOF

    # Build package
    dpkg-deb --build "$build_dir" "$DEB_DIR/pool/main/m/mtop/mtop_${VERSION}_${deb_arch}.deb"
}

# Create deb packages
create_deb "amd64" "amd64" "amd64"
create_deb "arm64" "arm64" "arm64"
create_deb "arm" "armhf" "armhf"

# Clean temporary build
rm -rf "$TMP_BUILD" mtop-amd64 mtop-arm64 mtop-armhf

echo "=== Generating APT Repository Structure ==="

# Define architectures
archs=("amd64" "arm64" "armhf")

for arch in "${archs[@]}"; do
    binary_dir="$DEB_DIR/dists/stable/main/binary-$arch"
    mkdir -p "$binary_dir"

    echo "Generating Packages for $arch..."
    # Generate Packages file relative to the repo root (which is the directory above pool)
    cd "$DEB_DIR"
    dpkg-scanpackages --arch "$arch" pool/main/m/mtop > "dists/stable/main/binary-$arch/Packages"
    gzip -9fk "dists/stable/main/binary-$arch/Packages"
    cd ..
done

echo "Generating Release file..."
cd "$DEB_DIR/dists/stable"
cat << EOF > Release
Origin: Mtop APT Repo
Label: Mtop
Suite: stable
Codename: stable
Architectures: amd64 arm64 armhf
Components: main
Description: Mtop APT Repository for Proxmox LXC & Bare Metal
EOF

# Append hashes of Package files to Release
echo "Date: $(date -Ru)" >> Release
echo "SHA256:" >> Release

# Calculate checksums
for arch in "${archs[@]}"; do
    for f in "main/binary-$arch/Packages" "main/binary-$arch/Packages.gz"; do
        if [ -f "$f" ]; then
            sha=$(sha256sum "$f" | cut -d' ' -f1)
            sz=$(wc -c < "$f")
            echo " $sha $sz $f" >> Release
        fi
    done
done
cd ../..

echo "APT Repository created successfully under '$(pwd)/$DEB_DIR'!"
