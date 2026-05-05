#!/bin/bash
# Bundle libmpv dylib into macOS .app bundle
# Usage: bundle-mpv-macos.sh <path/to/App.app>
set -e

APP="$1"
if [ -z "$APP" ] || [ ! -d "$APP" ]; then
    echo "Usage: $0 <path/to/App.app>"
    exit 1
fi

MACOS_DIR="$APP/Contents/MacOS"
FRAMEWORKS_DIR="$APP/Contents/Frameworks"
EXE="$MACOS_DIR/tagfile-gui"

if [ ! -f "$EXE" ]; then
    echo "Error: executable not found at $EXE"
    exit 1
fi

# Find libmpv from Homebrew (Apple Silicon or Intel)
MPV_LIB=""
for dir in /opt/homebrew/lib /usr/local/lib /opt/homebrew/opt/mpv/lib; do
    if [ -f "$dir/libmpv.2.dylib" ]; then
        MPV_LIB="$dir"
        break
    fi
done

if [ -z "$MPV_LIB" ]; then
    echo "Error: libmpv.2.dylib not found. Install mpv: brew install mpv"
    exit 1
fi

echo "Found libmpv at $MPV_LIB"

# Create Frameworks directory
mkdir -p "$FRAMEWORKS_DIR"

# Copy libmpv dylibs
cp "$MPV_LIB/libmpv.2.dylib" "$FRAMEWORKS_DIR/"
cp "$MPV_LIB/libmpv.dylib" "$FRAMEWORKS_DIR/"

# Fix install names
install_name_tool -change "$MPV_LIB/libmpv.2.dylib" \
    "@executable_path/../Frameworks/libmpv.2.dylib" "$EXE"

echo "Done: libmpv bundled into $APP"
