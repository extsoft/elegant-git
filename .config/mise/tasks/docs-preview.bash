#!/usr/bin/env bash
#MISE description="preview the user documentation locally"
set -o errexit -o nounset

root="$(git rev-parse --show-toplevel)"
cd "${root}"

port="${DOCS_PORT:-4000}"

docker build -t elegant-git-docs -f docs/Dockerfile docs
echo "Preview: http://127.0.0.1:${port}"
docker run --rm -it \
  -p "${port}:4000" \
  -v "${PWD}/docs:/srv/docs" \
  elegant-git-docs
