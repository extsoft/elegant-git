#!/usr/bin/env bash
set -euo pipefail

usage() {
    cat <<'EOF'
Usage: verify-task.sh

Runs project verification from repo root: mise run fix, then mise run test
when Go sources exist. Exits non-zero on failure.
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    usage
    exit 0
fi

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [[ -z "$repo_root" ]]; then
    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    repo_root="$(cd "$script_dir/../../../../.." && pwd)"
fi

cd "$repo_root"

echo "verify-task.sh: mise run fix"
mise run fix

if [[ -f go.mod ]] && [[ -d cmd || -d internal ]]; then
    echo "verify-task.sh: mise run test"
    mise run test
fi

echo "verify-task.sh: OK"
