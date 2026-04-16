#!/bin/bash

# Release packaging script
# Usage: ./hack/release.sh [version]

set -e

VERSION="${1:-v3.0.0}"
VERSION_NUM="${VERSION#v}"

echo "=== Packaging release $VERSION ==="

# Clean and create release directory
rm -rf release
mkdir -p release

# Package each platform
for dir in bin/$VERSION/*/; do
    if [ -d "$dir" ]; then
        platform=$(basename "$dir")
        os=$(echo "$platform" | sed 's/_.*//')
        arch=$(echo "$platform" | sed 's/.*_//')
        
        binary="sync-canal-go"
        if [ "$os" = "windows" ]; then
            binary="sync-canal-go.exe"
        fi
        
        if [ -f "$dir$binary" ]; then
            echo "Packaging $os/$arch..."
            pkgname="sync-canal-go_${VERSION_NUM}_${os}_${arch}"
            
            mkdir -p "release/$pkgname"
            cp "$dir$binary" "release/$pkgname/"
            cp -r manifest "release/$pkgname/"
            cp README.md "release/$pkgname/" 2>/dev/null || true
            
            if [ "$os" = "windows" ]; then
                (cd release && zip -r "$pkgname.zip" "$pkgname")
            else
                (cd release && tar -czf "$pkgname.tar.gz" "$pkgname")
            fi
            
            rm -rf "release/$pkgname"
        else
            echo "WARNING: Binary not found: $dir$binary"
        fi
    fi
done

# Generate checksums
echo "=== Generating checksums ==="
(cd release && shasum -a 256 *.tar.gz *.zip > checksums.txt 2>/dev/null || sha256sum *.tar.gz *.zip > checksums.txt)

echo "=== Done! Release packages ==="
ls -lh release/
