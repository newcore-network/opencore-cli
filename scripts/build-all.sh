#!/bin/bash
# Build script for all platforms

set -e

VERSION=${1:-"dev"}
OUTPUT_DIR="build"

echo "🏗️  Building OpenCore CLI v${VERSION} for all platforms..."
echo ""

# Clean output directory
rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"

# Build for each platform
platforms=(
    "windows/amd64"
    "darwin/amd64"
    "darwin/arm64"
    "linux/amd64"
)

for platform in "${platforms[@]}"; do
    GOOS="${platform%/*}"
    GOARCH="${platform#*/}"
    
    output_name="opencore-${GOOS}-${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
        output_name="${output_name}.exe"
    fi
    
    echo "📦 Building for ${GOOS}/${GOARCH}..."
    
    GOOS="$GOOS" GOARCH="$GOARCH" CGO_ENABLED=0 \
        go build -ldflags "-X main.version=${VERSION}" \
        -o "${OUTPUT_DIR}/${output_name}" \
        ./cmd/opencore
    
    echo "✅ ${output_name}"
    echo ""
done

echo "✨ Build complete! Binaries in ${OUTPUT_DIR}/"
ls -lh "$OUTPUT_DIR"

