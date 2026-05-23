#!/usr/bin/env bash
#MISE description="build and install git-elegant to PATH"
set -o errexit -o nounset

: "${DIST_DIR:=dist}"
: "${INSTALL_DIR:=${HOME}/.local/bin}"
root="$(git rev-parse --show-toplevel)"
cd "${root}"

mise run build

artifact="${DIST_DIR}/git-elegant"

install_git_elegant() {
  local install_dir="${1}"
  local target="${install_dir}/git-elegant"
  mkdir -p "${install_dir}"
  install -m 755 "${artifact}" "${target}"
  if ! "${target}" version >/dev/null 2>&1; then
    echo "install: ${target} cannot run here (exit $? — often signal 9 / macOS blocking this path)." >&2
    return 1
  fi
  echo "Installed: ${target}"
}

if install_git_elegant "${INSTALL_DIR}"; then
  exit 0
fi
if [[ ${INSTALL_DIR} == "${HOME}/.local/bin" ]]; then
  echo "install: retrying ${HOME}/bin" >&2
  install_git_elegant "${HOME}/bin"
else
  exit 1
fi
