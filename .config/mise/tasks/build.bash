#!/usr/bin/env bash
#MISE description="build the git-elegant binary"
set -o errexit -o nounset

: "${DIST_DIR:=dist}"
root="$(git rev-parse --show-toplevel)"
cd "${root}"

if [[ ! -f cmd/git-elegant/main.go ]]; then
  echo "build: cmd/git-elegant/main.go not found (complete task 0002-cli-skeleton)"
  exit 1
fi

go mod tidy
mkdir -p "${DIST_DIR}"
version="$(git describe --tags --always --abbrev=7 2>/dev/null || echo dev)"
go build \
  -ldflags="-s -w -X github.com/extsoft/elegant-git/internal/version.Version=${version}" \
  -o "${DIST_DIR}/git-elegant" \
  ./cmd/git-elegant
echo "Artifact: ${DIST_DIR}/git-elegant"
