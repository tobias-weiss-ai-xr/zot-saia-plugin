#!/usr/bin/env bash
set -euo pipefail

VERSION="${VERSION:-1.4.0}"
BINARY="zot-saia-plugin"
GO="${GO:-$(command -v go)}"

echo "==> building ${BINARY} v${VERSION}"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 "$GO" build \
    -ldflags "-s -w -X main.version=${VERSION}" \
    -o "${BINARY}" ./cmd/zot-saia-plugin
echo "==> built ./${BINARY}"
