#!/bin/bash

# NetCrate Build Script
# This script builds NetCrate for multiple platforms

set -e

PROJECT_NAME="netcrate"
VERSION=${VERSION:-"1.0.0"}
BUILD_DIR="dist"
CMD_PATH="./cmd/netcrate"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[BUILD]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Create build directory
log "Creating build directory: $BUILD_DIR"
mkdir -p "$BUILD_DIR"

# Build information
BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS="-w -s"
LDFLAGS="$LDFLAGS -X main.Version=$VERSION"
LDFLAGS="$LDFLAGS -X main.BuildTime=$BUILD_TIME"
LDFLAGS="$LDFLAGS -X main.GitCommit=$GIT_COMMIT"

# Platform configurations
declare -a PLATFORMS=(
    "darwin/amd64"
    "darwin/arm64"
    "linux/amd64" 
    "linux/arm64"
    "windows/amd64"
)

# Build for each platform
for platform in "${PLATFORMS[@]}"; do
    IFS='/' read -r -a platform_split <<< "$platform"
    GOOS="${platform_split[0]}"
    GOARCH="${platform_split[1]}"
    
    output_name="$PROJECT_NAME"
    if [ "$GOOS" = "windows" ]; then
        output_name="$output_name.exe"
    fi
    
    output_path="$BUILD_DIR/${PROJECT_NAME}_${GOOS}_${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
        output_path="$output_path.exe"
    fi
    
    log "Building for $GOOS/$GOARCH..."
    
    # Skip if dependencies are not available
    if ! env GOOS="$GOOS" GOARCH="$GOARCH" go build -ldflags "$LDFLAGS" -o "$output_path" "$CMD_PATH" 2>/dev/null; then
        warn "Skipping $GOOS/$GOARCH (dependencies not available)"
        continue
    fi
    
    log "✓ Built: $output_path"
    
    # Create archive
    archive_name="${PROJECT_NAME}_${VERSION}_${GOOS}_${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
        archive_name="$archive_name.zip"
        log "Creating archive: $BUILD_DIR/$archive_name"
        (cd "$BUILD_DIR" && zip -q "$archive_name" "$(basename "$output_path")")
    else
        archive_name="$archive_name.tar.gz"
        log "Creating archive: $BUILD_DIR/$archive_name"
        (cd "$BUILD_DIR" && tar -czf "$archive_name" "$(basename "$output_path")")
    fi
done

# Generate checksums
log "Generating checksums..."
(cd "$BUILD_DIR" && shasum -a 256 *.tar.gz *.zip 2>/dev/null > checksums.txt) || true

# List build artifacts
log "Build complete! Artifacts:"
ls -la "$BUILD_DIR"

log "Build script completed successfully!"