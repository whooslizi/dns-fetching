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

cp dns-fetching AppDir/usr/bin/
cp dns-fetching-helper AppDir/usr/bin/
cp assets/dns-fetching.desktop AppDir/usr/share/applications/
cp assets/dns-fetching.desktop AppDir/

# use icon if it exists, otherwise create a placeholder
if [ -f assets/icon.png ]; then
    mkdir -p AppDir/usr/share/icons/hicolor/256x256/apps
    cp assets/icon.png AppDir/usr/share/icons/hicolor/256x256/apps/dns-fetching.png
    cp assets/icon.png AppDir/dns-fetching.png
    ICON_FLAG="-i AppDir/dns-fetching.png"
elif [ -f assets/icon.svg ]; then
    mkdir -p AppDir/usr/share/icons/hicolor/scalable/apps
    cp assets/icon.svg AppDir/usr/share/icons/hicolor/scalable/apps/dns-fetching.svg
    cp assets/icon.svg AppDir/dns-fetching.svg
    ICON_FLAG="-i AppDir/dns-fetching.svg"
else
    echo "Warning: no icon found in assets/, using placeholder SVG"
    mkdir -p AppDir/usr/share/icons/hicolor/scalable/apps
    echo '<svg xmlns="http://www.w3.org/2000/svg" width="256" height="256"></svg>' > AppDir/dns-fetching.svg
    cp AppDir/dns-fetching.svg AppDir/usr/share/icons/hicolor/scalable/apps/dns-fetching.svg
    ICON_FLAG="-i AppDir/dns-fetching.svg"
fi

# download LinuxDeploy if not present
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
    $ICON_FLAG \
    --output appimage

echo "=== Done! ==="
echo "AppImage: ${APP_NAME}-${ARCH}.AppImage"
