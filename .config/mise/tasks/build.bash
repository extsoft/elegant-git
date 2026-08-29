#!/usr/bin/env bash
#MISE description="build the eg binary"
set -o errexit -o nounset

: "${DIST_DIR:=dist}"
root="$(git rev-parse --show-toplevel)"
cd "${root}"

if [[ ! -f cmd/eg/main.go ]]; then
  echo "build: cmd/eg/main.go not found"
  exit 1
fi

go mod tidy
mkdir -p "${DIST_DIR}"
version="$(git describe --tags --always --abbrev=7 2>/dev/null || echo dev)"
go build \
  -ldflags="-s -w -X github.com/extsoft/elegant-git/internal/version.Version=${version}" \
  -o "${DIST_DIR}/eg" \
  ./cmd/eg
echo "Artifact: ${DIST_DIR}/eg"
