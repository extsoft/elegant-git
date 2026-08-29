#!/usr/bin/env bash
#MISE description="publish a GitHub release"
set -o pipefail -o errexit -o nounset

root="$(git rev-parse --show-toplevel)"
cd "${root}"

datever_prefix() {
  local y m d
  y=$(date +%Y)
  m=$((10#$(date +%m)))
  d=$((10#$(date +%d)))
  echo "v${y}.${m}.${d}"
}

git fetch --tags
echo "Release type:"
echo "  1) stable     — vYYYY.M.D"
echo "  2) prerelease — vYYYY.M.D.hhmm-prerelease"
new_version=""
while [[ -z $new_version ]]; do
  read -r -p "Choose [1/2]: " release_type
  case "$release_type" in
  1) new_version=$(datever_prefix) ;;
  2) new_version="$(datever_prefix).$(date +%H%M)-prerelease" ;;
  *) echo "Enter 1 for stable or 2 for prerelease." >&2 ;;
  esac
done
echo "New version: $new_version"

stable_re='^v[0-9]{4}\.[0-9]{1,2}\.[0-9]{1,2}$'
prerelease_re='^v[0-9]{4}\.[0-9]{1,2}\.[0-9]{1,2}\.[0-9]{4}-prerelease$'

if [[ ! $new_version =~ $stable_re ]] && [[ ! $new_version =~ $prerelease_re ]]; then
  echo "Error: version must match vYYYY.M.D or vYYYY.M.D.hhmm-prerelease" >&2
  exit 1
fi

if git rev-parse "refs/tags/${new_version}" >/dev/null 2>&1; then
  echo "Error: tag already exists locally: ${new_version}" >&2
  exit 1
fi

if [[ -n "$(git ls-remote origin "refs/tags/${new_version}")" ]]; then
  echo "Error: tag already exists on origin: ${new_version}" >&2
  exit 1
fi

BASE_TAG=""
while IFS= read -r t; do
  [[ -z $t ]] && continue
  [[ $t == *prerelease* ]] && continue
  BASE_TAG="$t"
  break
done < <(git tag --sort=-creatordate)

notes_file=$(mktemp)
trap 'rm -f "$notes_file"' EXIT

github_repo() {
  local o
  o=$(gh repo view --json nameWithOwner -q .nameWithOwner 2>/dev/null || true)
  if [[ -n $o ]]; then
    echo "$o"
    return
  fi
  local url
  url=$(git remote get-url origin)
  if [[ $url =~ github\.com:([^/]+/[^.]+)(\.git)?$ ]]; then
    echo "${BASH_REMATCH[1]}"
    return
  fi
  if [[ $url =~ github\.com/([^/]+/[^.]+)(\.git)?$ ]]; then
    echo "${BASH_REMATCH[1]}"
    return
  fi
  echo "Error: could not determine GitHub owner/repo (set origin or run gh from the repo)" >&2
  exit 1
}

repo=$(github_repo)

{
  first=1
  while IFS= read -r sha; do
    [[ -z $sha ]] && continue
    subject=$(git log -1 --format=%s "$sha")
    body=$(git log -1 --format=%b "$sha")
    if [[ -z $first ]]; then
      printf '\n\n'
    fi
    first=
    printf '### [%s](https://github.com/%s/commit/%s)\n\n' "$subject" "$repo" "$sha"
    if [[ -n $body ]]; then
      printf '%s\n' "$body"
    fi
  done < <(
    if [[ -n ${BASE_TAG:-} ]]; then
      git rev-list --reverse "${BASE_TAG}..HEAD"
    else
      git rev-list --reverse HEAD
    fi
  )
} >"$notes_file"

mise run release:artifacts "$new_version"

DIST_DIR="${DIST_DIR:-dist}"
release_files=()
while IFS= read -r -d '' f; do release_files+=("$f"); done < <(find "./${DIST_DIR}/$new_version" -type f -print0)

gh_args=(release create "$new_version" "${release_files[@]}" --title "$new_version" --notes-file "$notes_file")
if [[ $new_version =~ $prerelease_re ]]; then
  gh_args+=(--prerelease)
fi
gh "${gh_args[@]}"
