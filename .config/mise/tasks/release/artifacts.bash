#!/usr/bin/env bash
#MISE description="build release artifacts"
set -o pipefail -o errexit -o nounset

BINARY="git-elegant"
MODULE="github.com/extsoft/elegant-git"

PLATFORMS=(
  "linux-amd64"
  "linux-arm64"
  "linux-386"
  "windows-amd64"
  "windows-arm64"
  "windows-386"
  "darwin-amd64"
  "darwin-arm64"
)

# Variants keep the binaries compatible with the oldest CPUs of each architecture.
get_build_variants() {
  local GOARCH=$1

  local GOAMD64=""
  local GO386=""
  local GOARM64=""

  case "$GOARCH" in
  amd64)
    GOAMD64="v1"
    ;;
  386)
    GO386="sse2"
    ;;
  arm64)
    GOARM64="v8.0"
    ;;
  esac

  echo "$GOAMD64|$GO386|$GOARM64"
}

generate_checksum() {
  local archive_path=$1
  local checksum_file="${archive_path}.sha256"

  if command -v sha256sum >/dev/null; then
    sha256sum "$archive_path" | cut -d' ' -f1 >"$checksum_file"
  elif command -v shasum >/dev/null; then
    shasum -a 256 "$archive_path" | cut -d' ' -f1 >"$checksum_file"
  else
    echo "Error: Neither sha256sum nor shasum found" >&2
    exit 1
  fi

  echo "Generated checksum: $checksum_file"
}

build_and_archive_platform() {
  local dist_dir=$1
  local version=$2
  local platform=$3
  IFS='-' read -r GOOS GOARCH <<<"$platform"
  local archive_name="${BINARY}-${version}-${platform}"
  local staging_dir="${dist_dir}/${archive_name}"
  mkdir -p "$staging_dir"

  IFS='|' read -r GOAMD64 GO386 GOARM64 <<<"$(get_build_variants "$GOARCH")"

  local binary_name="$BINARY"
  if [ "$GOOS" = "windows" ]; then
    binary_name="${BINARY}.exe"
  fi

  echo "Building $platform..."

  export CGO_ENABLED=0
  export GOOS="$GOOS"
  export GOARCH="$GOARCH"
  [ -n "$GOAMD64" ] && export GOAMD64="$GOAMD64"
  [ -n "$GO386" ] && export GO386="$GO386"
  [ -n "$GOARM64" ] && export GOARM64="$GOARM64"

  go build \
    -ldflags="-s -w -X ${MODULE}/internal/version.Version=${version}" \
    -o "$staging_dir/$binary_name" \
    ./cmd/git-elegant

  cp LICENSE "$staging_dir/"
  cp README.md "$staging_dir/"

  local archive_path
  archive_path=$(cd "$dist_dir" && pwd)/"${archive_name}"
  pushd "$staging_dir" >/dev/null
  if [ "$GOOS" = "windows" ]; then
    zip -r "${archive_path}.zip" . >/dev/null
    echo "Created ${archive_path}.zip"
    generate_checksum "${archive_path}.zip"
  else
    # COPYFILE_DISABLE=1 keeps macOS from adding ._ resource forks to the tarball
    COPYFILE_DISABLE=1 tar -czf "${archive_path}.tar.gz" .
    echo "Created ${archive_path}.tar.gz"
    generate_checksum "${archive_path}.tar.gz"
  fi
  popd >/dev/null

  rm -rf "$staging_dir"
}

root="$(git rev-parse --show-toplevel)"
cd "${root}"

VERSION=${1:?version is required}
echo "Building release artifacts for version: $VERSION"
DIST_DIR="${DIST_DIR:-dist}"
DIST_VERSION_DIR="$DIST_DIR/$VERSION"
echo "Output directory: ${DIST_VERSION_DIR}"
if [ -d "${DIST_VERSION_DIR}" ]; then
  echo "Removing directory: ${DIST_VERSION_DIR}"
  rm -rf "${DIST_VERSION_DIR}"
fi
mkdir -p "${DIST_VERSION_DIR}"

for platform in "${PLATFORMS[@]}"; do
  build_and_archive_platform "${DIST_VERSION_DIR}" "${VERSION}" "$platform"
done

cp install.sh "${DIST_VERSION_DIR}/"
echo "Created ${DIST_VERSION_DIR}/install.sh"

echo "Release artifacts built successfully in ${DIST_VERSION_DIR}"
