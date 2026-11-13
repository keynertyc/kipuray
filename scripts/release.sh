#!/bin/sh
set -eu
version="${1:-dev}"
mkdir -p dist
for platform in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64; do
  target_os="${platform%/*}"
  target_arch="${platform#*/}"
  extension=""
  if [ "$target_os" = windows ]; then extension=.exe; fi
  name="kipuray_${version}_${target_os}_${target_arch}${extension}"
  CGO_ENABLED=0 GOOS="$target_os" GOARCH="$target_arch" go build -trimpath -ldflags "-s -w -X main.version=$version" -o "dist/$name" ./cmd/kipuray
done
cp LICENSE NOTICE THIRD_PARTY_LICENSES.txt dist/
cd dist
if command -v sha256sum >/dev/null 2>&1; then sha256sum kipuray_* > SHA256SUMS; else shasum -a 256 kipuray_* > SHA256SUMS; fi
