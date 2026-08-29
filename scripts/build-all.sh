#!/usr/bin/env bash

set -euo pipefail

VERSION=${1:-$(git describe --tags --always --dirty 2>/dev/null || printf 'dev')}
OUTPUT_DIR=${OUTPUT_DIR:-build}

printf 'Building OpenCore CLI %s for all platforms...\n' "$VERSION"

# Clean output directory
rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"

# Build for each platform
platforms=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

for platform in "${platforms[@]}"; do
    GOOS="${platform%/*}"
    GOARCH="${platform#*/}"
    
    output_name="opencore-${GOOS}-${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
        output_name="${output_name}.exe"
    fi
    
    printf 'Building for %s/%s...\n' "$GOOS" "$GOARCH"
    
    GOOS="$GOOS" GOARCH="$GOARCH" CGO_ENABLED=0 \
        go build -ldflags "-X main.version=${VERSION}" \
        -o "${OUTPUT_DIR}/${output_name}" \
        .
    
    printf 'Built %s\n' "$output_name"
done

printf 'Build complete: %s/\n' "$OUTPUT_DIR"
