#!/usr/bin/env bash
set -euo pipefail

usage() {
    cat <<'EOF'
Usage: validate-folder.sh <.tasks/folder>

Checks plan.md exists and each NNNN-*.md has required frontmatter keys.
Exit 1 on first failure.
EOF
}

folder="${1:-}"
if [[ -z "$folder" || ! -d "$folder" ]]; then
    echo "validate-folder.sh: missing or invalid folder: $folder" >&2
    usage
    exit 1
fi

if [[ ! -f "$folder/plan.md" ]]; then
    echo "validate-folder.sh: missing plan.md in $folder" >&2
    exit 1
fi

shopt -s nullglob
task_files=("$folder"/[0-9][0-9][0-9][0-9]-*.md)
if [[ ${#task_files[@]} -eq 0 ]]; then
    echo "validate-folder.sh: no NNNN-*.md task files in $folder" >&2
    exit 1
fi

required_keys=(id title status depends_on acceptance github)
fail=0

for f in "${task_files[@]}"; do
    base="$(basename "$f")"
    for key in "${required_keys[@]}"; do
        if ! grep -q "^${key}:" "$f" 2>/dev/null; then
            echo "validate-folder.sh: $base missing frontmatter key: $key" >&2
            fail=1
        fi
    done
done

if [[ "$fail" -ne 0 ]]; then
    exit 1
fi

echo "validate-folder.sh: OK ($folder, ${#task_files[@]} tasks)"
