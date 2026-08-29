#!/usr/bin/env bash
#MISE description="build and install eg to PATH"
set -o errexit -o nounset

: "${DIST_DIR:=dist}"
: "${INSTALL_DIR:=${HOME}/.local/bin}"
root="$(git rev-parse --show-toplevel)"
cd "${root}"

mise run build

artifact="${DIST_DIR}/eg"

remove_legacy() {
  local dir="${1}"
  local legacy="${dir}/git-elegant"
  if [[ -e ${legacy} ]]; then
    rm -f "${legacy}"
    echo "Removed leftover: ${legacy}"
  fi
}

install_eg() {
  local install_dir="${1}"
  local target="${install_dir}/eg"
  mkdir -p "${install_dir}"
  install -m 755 "${artifact}" "${target}"
  if ! "${target}" version >/dev/null 2>&1; then
    echo "install: ${target} cannot run here (exit $? — often signal 9 / macOS blocking this path)." >&2
    return 1
  fi
  echo "Installed: ${target}"
}

remove_legacy_defaults() {
  remove_legacy "${DIST_DIR}"
  remove_legacy "${INSTALL_DIR}"
  remove_legacy "${HOME}/.local/bin"
  remove_legacy "${HOME}/bin"
}

if install_eg "${INSTALL_DIR}"; then
  remove_legacy_defaults
  exit 0
fi
if [[ ${INSTALL_DIR} == "${HOME}/.local/bin" ]]; then
  echo "install: retrying ${HOME}/bin" >&2
  install_eg "${HOME}/bin"
  remove_legacy_defaults
else
  exit 1
fi
