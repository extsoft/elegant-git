#!/usr/bin/env bash
set -euo pipefail

usage() {
    cat <<'EOF'
Usage: new-request.sh <slug> [task-count]

Creates .tasks/<slug>-<shorthash>/ with plan.md and numbered task skeletons.
Exits with error if the target folder already exists.

  slug        kebab-case request name (e.g. auth-rewrite)
  task-count  number of task files to create (default: 3)
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" || $# -lt 1 ]]; then
    usage
    exit "${1:+0}"
fi

slug="$1"
task_count="${2:-3}"

if ! [[ "$slug" =~ ^[a-z0-9]+(-[a-z0-9]+)*$ ]]; then
    echo "new-request.sh: slug must be lowercase kebab-case: $slug" >&2
    exit 1
fi

if ! [[ "$task_count" =~ ^[0-9]+$ ]] || [[ "$task_count" -lt 1 ]]; then
    echo "new-request.sh: task-count must be a positive integer" >&2
    exit 1
fi

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [[ -z "$repo_root" ]]; then
    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    repo_root="$(cd "$script_dir/../../../../.." && pwd)"
fi

cd "$repo_root"

if command -v openssl >/dev/null 2>&1; then
    shorthash="$(openssl rand -hex 2)"
else
    shorthash="$(printf '%04x' $((RANDOM * RANDOM % 65536)))"
fi

folder=".tasks/${slug}-${shorthash}"

if [[ -e "$folder" ]]; then
    echo "new-request.sh: folder already exists: $folder" >&2
    exit 1
fi

mkdir -p "$folder"

cat >"$folder/plan.md" <<EOF
# ${slug}

## Goal

(TODO: what done means)

## Non-goals

- (TODO)

## Decision (ADR-lite)

**Chosen:** (TODO)

**Rejected:**

- (TODO) — because (TODO)

**Consequences:** (TODO)

## Risks and rollback

(TODO)

## Tasks overview

EOF

for ((i = 1; i <= task_count; i++)); do
    id="$(printf '%04d' "$i")"
    task_file="${folder}/${id}-task-${i}.md"
    cat >>"$folder/plan.md" <<EOF
${i}. [${id}-task-${i}.md](${id}-task-${i}.md) — (TODO summary)
EOF
    cat >"$task_file" <<EOF
---
id: "${id}"
title: "TODO task ${i}"
status: pending
depends_on: []
acceptance:
  - "TODO verifiable outcome"
github:
  repo: null
  issue: null
---

## Context

See [plan.md](plan.md).

## Steps

1. (TODO)

## Definition of Done

- See [.agents/references/definition-of-done.md](../../.agents/references/definition-of-done.md)
- Plus \`acceptance\` items in frontmatter
EOF
done

echo "$folder"
