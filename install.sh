#!/usr/bin/env sh
set -o errexit -o nounset

REPOSITORY="https://github.com/extsoft/elegant-git"
BINARY="git-elegant"
BINDIR="${BINDIR:-}"
TAG=""

usage() {
  this="$1"
  cat <<EOF
$this [-d bindir] [-v tag] [-x]

  -d sets install directory (default: first writable of ~/.local/bin, ~/bin, /usr/local/bin, /usr/bin, /usr/sbin, /opt/bin; else you are prompted)
  -v sets version tag from ${REPOSITORY}/releases
     If -v is not set, the latest release is used.
  -x turns on debug logging

EOF
  exit 2
}

log_priority=6
log_set_priority() { log_priority="$1"; }
log_info() {
  [ "$log_priority" -ge 6 ] || return 0
  echo "$@" 1>&2
}
log_step() {
  [ "$log_priority" -ge 6 ] || return 0
  echo "  → $*" 1>&2
}
log_debug() {
  [ "$log_priority" -ge 7 ] || return 0
  echo "  [debug] $*" 1>&2
}
log_err() {
  echo "  ✗ $*" 1>&2
}

parse_args() {
  while getopts "d:hv:x?" arg; do
    case "$arg" in
    d) BINDIR="$OPTARG" ;;
    v) TAG="$OPTARG" ;;
    x) log_set_priority 7 ;;
    h | \?) usage "$0" ;;
    esac
  done
  shift $((OPTIND - 1))
  [ -z "$TAG" ] && TAG="${1:-}"
}

default_dest_dir() {
  for dir in "${HOME}/.local/bin" "${HOME}/bin" "/usr/local/bin" "/usr/bin" "/usr/sbin" "/opt/bin"; do
    if [ -d "$dir" ] && [ -w "$dir" ]; then
      log_step "  $dir: exists, writable (using)"
      echo "$dir"
      return 0
    fi
    if [ -d "$dir" ]; then
      log_err "  $dir: exists, not writable (skip)"
    else
      log_err "  $dir: missing (skip)"
    fi
  done
  return 1
}

uname_os() {
  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  case "$os" in
  msys* | mingw* | cygwin* | win*) os="windows" ;;
  sunos) if [ "$(uname -o)" = "illumos" ]; then os="illumos"; fi ;;
  esac
  echo "$os"
}

uname_arch() {
  arch=$(uname -m)
  case "$arch" in
  x86_64 | x64) arch="amd64" ;;
  x86 | i686 | i386) arch="386" ;;
  aarch64 | arm64) arch="arm64" ;;
  esac
  echo "$arch"
}

uname_os_check() {
  os=$(uname_os)
  case "$os" in
  darwin | linux | windows) return 0 ;;
  esac
  log_err "Your OS ($os) is not supported. See ${REPOSITORY}/releases for binaries."
  return 1
}

uname_arch_check() {
  arch=$(uname_arch)
  case "$arch" in
  386 | amd64 | arm64) return 0 ;;
  esac
  log_err "Your architecture ($arch) is not supported. See ${REPOSITORY}/releases"
  return 1
}

is_command() {
  command -v "$1" >/dev/null 2>&1
}

http_download_curl() {
  local_file="$1"
  source_url="$2"
  header="${3:-}"
  log_debug "http_download_curl $source_url"
  if [ -n "$header" ]; then
    code=$(curl -w '%{http_code}' -sSL -H "$header" -o "$local_file" "$source_url")
  else
    code=$(curl -w '%{http_code}' -sSL -o "$local_file" "$source_url")
  fi
  [ "$code" = "200" ] || return 1
  return 0
}

http_download_wget() {
  local_file="$1"
  source_url="$2"
  header="${3:-}"
  log_debug "http_download_wget $source_url"
  if [ -n "$header" ]; then
    wget -q --header="$header" -O "$local_file" "$source_url" || return 1
  else
    wget -q -O "$local_file" "$source_url" || return 1
  fi
  return 0
}

http_download() {
  if is_command curl; then
    http_download_curl "$@"
    return $?
  fi
  if is_command wget; then
    http_download_wget "$@"
    return $?
  fi
  log_err "This script needs curl or wget to download files."
  return 1
}

http_copy() {
  tmp=$(mktemp)
  http_download "$tmp" "$1" "$2" || return 1
  body=$(cat "$tmp")
  rm -f "$tmp"
  echo "$body"
}

github_release_tag() {
  version="$1"
  if test -z "$version"; then
    url="${REPOSITORY}/releases/latest"
  else
    url="${REPOSITORY}/releases/tags/${version}"
  fi
  json=$(http_copy "$url" "Accept: application/json") || return 1
  test -z "$json" && return 1
  case "$json" in
  \{*) ;;
  *)
    log_err "No release found. See ${REPOSITORY}/releases"
    return 1
    ;;
  esac
  tag=$(echo "$json" | tr -s '\n' ' ' | sed 's/.*"tag_name":"//' | sed 's/".*//')
  case "$tag" in
  *[!a-zA-Z0-9._-]* | "")
    log_err "Invalid version from GitHub"
    return 1
    ;;
  esac
  echo "$tag"
}

hash_sha256() {
  if is_command gsha256sum; then
    gsha256sum "$1" | cut -d' ' -f1
  elif is_command sha256sum; then
    sha256sum "$1" | cut -d' ' -f1
  elif is_command shasum; then
    shasum -a 256 "$1" | cut -d' ' -f1
  else
    log_err "Checksum verification requires sha256sum or shasum (not found)"
    return 1
  fi
}

hash_sha256_verify() {
  target="$1"
  want="$2"
  got=$(hash_sha256 "$target") || return 1
  [ "$want" = "$got" ] || return 1
  return 0
}

untar() {
  tarball="$1"
  _tar_stderr=$(mktemp)
  case "$tarball" in
  *.tar.gz | *.tgz)
    tar -xzf "$tarball" 2>"$_tar_stderr"
    _code=$?
    grep -v 'Ignoring unknown extended header' "$_tar_stderr" >&2 || true
    rm -f "$_tar_stderr"
    return $_code
    ;;
  *.zip) unzip -q "$tarball" ;;
  *)
    log_err "Unsupported archive format"
    return 1
    ;;
  esac
}

execute() {
  tmpdir=$(mktemp -d)
  trap 'rm -rf "$tmpdir"' EXIT
  log_debug "downloading into $tmpdir"

  log_info "Installing Elegant Git"
  log_info ""

  log_step "Resolving version..."
  if [ -z "$TAG" ]; then
    log_debug "fetching latest release"
  else
    log_debug "fetching tag $TAG"
  fi
  TAG=$(github_release_tag "$TAG") || {
    log_err "Could not find a release. See ${REPOSITORY}/releases"
    return 1
  }
  log_step "Using version $TAG"
  log_info ""

  os=$(uname_os)
  arch=$(uname_arch)
  log_step "Platform: $os / $arch"
  log_debug "os=$os arch=$arch tag=$TAG"
  log_info ""

  case "$os" in
  windows)
    archive_name="${BINARY}-${TAG}-${os}-${arch}.zip"
    ;;
  *)
    archive_name="${BINARY}-${TAG}-${os}-${arch}.tar.gz"
    ;;
  esac

  base_url="${REPOSITORY}/releases/download/${TAG}"
  archive_url="${base_url}/${archive_name}"
  checksum_url="${base_url}/${archive_name}.sha256"

  log_step "Downloading $archive_name..."
  http_download "${tmpdir}/${archive_name}" "$archive_url" || {
    log_err "Download failed. Check your connection and ${REPOSITORY}/releases"
    return 1
  }
  log_step "Downloaded"
  log_info ""

  log_step "Verifying checksum..."
  http_download "${tmpdir}/${archive_name}.sha256" "$checksum_url" || {
    log_err "Could not download checksum file"
    return 1
  }
  want_hash=$(tr -d '\n\r ' <"${tmpdir}/${archive_name}.sha256")
  hash_sha256_verify "${tmpdir}/${archive_name}" "$want_hash" || {
    log_err "Checksum verification failed. Aborting for safety."
    return 1
  }
  log_step "Checksum OK"
  log_info ""

  log_step "Installing to $BINDIR"
  if ! [ -d "$BINDIR" ]; then
    log_err "Install directory does not exist: $BINDIR"
    log_err "Create it first, e.g.: mkdir -p $BINDIR"
    return 1
  fi
  if ! [ -w "$BINDIR" ]; then
    log_err "Install directory is not writable: $BINDIR"
    return 1
  fi
  (cd "$tmpdir" && untar "$archive_name")
  if [ "$os" = "windows" ]; then
    install "${tmpdir}/${BINARY}.exe" "$BINDIR/"
    log_step "Installed ${BINARY}.exe"
  else
    install "${tmpdir}/${BINARY}" "$BINDIR/"
    log_step "Installed ${BINARY}"
  fi
  log_info ""

  log_info "Elegant Git $TAG is installed to $BINDIR"
  suggest_path_add
  log_info ""
  log_info "  Then configure Git and a repository:"
  log_info ""
  log_info "    git elegant git configure"
  log_info "    git elegant repo configure"
}

path_contains_dir() {
  dir="$1"
  case ":${PATH}:" in
  *":${dir}:"* | *":${dir}/"*) return 0 ;;
  *) return 1 ;;
  esac
}

suggest_path_add() {
  bindir_expanded="$BINDIR"
  case "$BINDIR" in
  ~/*) bindir_expanded="${HOME}/${BINDIR#\~/*}" ;;
  ~) bindir_expanded="$HOME" ;;
  ./*) bindir_expanded="$(cd "$BINDIR" 2>/dev/null && pwd)" ;;
  esac
  if path_contains_dir "$bindir_expanded" || path_contains_dir "$BINDIR"; then
    return 0
  fi
  log_info "  Add to PATH so that Git can find the ${BINARY} subcommand:"
  log_info ""
  case "$BINDIR" in
  /*)
    log_info "    export PATH=\"$BINDIR:\$PATH\""
    ;;
  *)
    bindir_abs=$(cd "$BINDIR" 2>/dev/null && pwd)
    if [ -n "$bindir_abs" ]; then
      log_info "    export PATH=\"$bindir_abs:\$PATH\""
    else
      log_info "    export PATH=\"$BINDIR:\$PATH\""
    fi
    ;;
  esac
  log_info ""
  log_info "  To make it permanent, add the line above to your shell config:"
  log_info "    bash: ~/.bashrc or ~/.profile"
  log_info "    zsh:  ~/.zshrc"
}

prompt_install_dir() {
  log_info "No standard install directory found (tried ~/.local/bin, ~/bin, /usr/local/bin, ...)."
  log_info ""
  printf "  Enter install directory [./bin]: " 1>&2
  read -r entered
  entered=$(echo "$entered" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
  if [ -n "$entered" ]; then
    echo "$entered"
  else
    echo "./bin"
  fi
}

main() {
  parse_args "$@"
  if [ -z "$BINDIR" ]; then
    log_info "Checking install directories"
    BINDIR=$(default_dest_dir) || true
    if [ -z "$BINDIR" ]; then
      if [ -t 0 ]; then
        BINDIR=$(prompt_install_dir)
        log_info ""
      else
        log_err "No install directory found. Use -d to specify one, e.g.: -d ~/.local/bin"
        exit 1
      fi
    fi
  fi
  if [ -z "$BINDIR" ]; then
    log_err "Install directory is required."
    exit 1
  fi
  log_info "Install directory: $BINDIR"
  log_info ""
  uname_os_check || exit 1
  uname_arch_check || exit 1
  execute
}

main "$@"
