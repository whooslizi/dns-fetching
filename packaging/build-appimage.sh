#!/bin/bash
set -euo pipefail

APP_NAME="DNS_Fetching"
ARCH="x86_64"

echo "=== Building DNS Fetching AppImage ==="

# build both binaries
echo "Building main binary..."
CGO_ENABLED=1 go build -ldflags="-s -w" -o dns-fetching .

echo "Building helper..."
CGO_ENABLED=1 go build -ldflags="-s -w" -o dns-fetching-helper ./cmd/helper/

# set up AppDir
echo "Setting up AppDir..."
rm -rf AppDir
mkdir -p AppDir/usr/bin
mkdir -p AppDir/usr/share/applications
mkdir -p AppDir/usr/share/icons/hicolor/256x256/apps

cp dns-fetching AppDir/usr/bin/
cp dns-fetching-helper AppDir/usr/bin/
cp assets/dns-fetching.desktop AppDir/usr/share/applications/
cp assets/dns-fetching.desktop AppDir/

# use icon if it exists, otherwise create a placeholder
if [ -f assets/icon.png ]; then
    cp assets/icon.png AppDir/usr/share/icons/hicolor/256x256/apps/dns-fetching.png
    cp assets/icon.png AppDir/dns-fetching.png
else
    echo "Warning: no icon.png found in assets/, using placeholder"
    # create a simple 1x1 PNG as placeholder
    printf '\x89PNG\r\n\x1a\n' > AppDir/dns-fetching.png
fi

# download linuxdeploy if not present
if [ ! -f linuxdeploy-x86_64.AppImage ]; then
    echo "Downloading linuxdeploy..."
    curl -fsSL -o linuxdeploy-x86_64.AppImage \
        "https://github.com/linuxdeploy/linuxdeploy/releases/download/continuous/linuxdeploy-x86_64.AppImage"
    chmod +x linuxdeploy-x86_64.AppImage
fi

# build the AppImage
echo "Packaging AppImage..."
./linuxdeploy-x86_64.AppImage \
    --appdir AppDir \
    -e AppDir/usr/bin/dns-fetching \
    -d AppDir/usr/share/applications/dns-fetching.desktop \
    -i AppDir/usr/share/icons/hicolor/256x256/apps/dns-fetching.png \
    --output appimage

echo "=== Done! ==="
echo "AppImage: ${APP_NAME}-${ARCH}.AppImage"
