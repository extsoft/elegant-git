#!/usr/bin/env bash
#MISE description="build a Linux eg and open a fresh Docker shell"
set -o errexit -o nounset

: "${DIST_DIR:=dist}"
root="$(git rev-parse --show-toplevel)"
cd "${root}"

if ! command -v docker >/dev/null; then
  echo "eg-preview: docker is required" >&2
  exit 1
fi

arch="$(docker version --format '{{.Server.Arch}}')"
case "${arch}" in
amd64 | x86_64) goarch=amd64 ;;
arm64 | aarch64) goarch=arm64 ;;
*)
  echo "eg-preview: unsupported docker arch: ${arch}" >&2
  exit 1
  ;;
esac

if [[ ! -f cmd/eg/main.go ]]; then
  echo "eg-preview: cmd/eg/main.go not found" >&2
  exit 1
fi

mkdir -p "${DIST_DIR}"
version="$(git describe --tags --always --abbrev=7 2>/dev/null || echo dev)"
CGO_ENABLED=0 GOOS=linux GOARCH="${goarch}" go build \
  -ldflags="-s -w -X github.com/extsoft/elegant-git/internal/version.Version=${version}" \
  -o "${DIST_DIR}/eg-linux" \
  ./cmd/eg
chmod 755 "${DIST_DIR}/eg-linux"

docker build -t elegant-git-preview -f .config/docker/eg-preview.Dockerfile .config/docker

echo "Artifact: ${DIST_DIR}/eg-linux  (linux/${goarch})"
docker run --rm -it \
  -v "${PWD}/${DIST_DIR}/eg-linux:/usr/local/bin/eg:ro" \
  elegant-git-preview
