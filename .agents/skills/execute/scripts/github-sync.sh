#!/usr/bin/env bash
set -euo pipefail

usage() {
    cat <<'EOF'
Usage: github-sync.sh <task-file.md>

Syncs task status to GitHub issues (stub: no-op until configured).
See .agents/references/adapters.md for the future contract.
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    usage
    exit 0
fi

task_file="${1:-}"

if [[ -z "$task_file" ]]; then
    echo "github-sync.sh: missing task file path" >&2
    usage
    exit 1
fi

if [[ ! -f "$task_file" ]]; then
    echo "github-sync.sh: file not found: $task_file" >&2
    exit 1
fi

echo "github-sync: adapter not configured; skipping ($task_file)"
exit 0
