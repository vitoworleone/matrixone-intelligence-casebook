#!/usr/bin/env bash
set -euo pipefail

readonly upstream_url="${MOI_PROTOTYPE_REPO:-https://github.com/matrixorigin/moi-prototype.git}"
readonly upstream_branch="${MOI_PROTOTYPE_BRANCH:-main}"
readonly upstream_prefix="html"
readonly target_prefix="product/moi-platform-prototype"
mode="sync"

usage() {
  printf 'Usage: %s [--check]\n' "${0##*/}"
  printf '\nEnvironment overrides:\n'
  printf '  MOI_PROTOTYPE_REPO    Upstream URL or local repository path\n'
  printf '  MOI_PROTOTYPE_BRANCH  Upstream branch (default: main)\n'
}

die() {
  printf 'error: %s\n' "$*" >&2
  exit 2
}

case "${1:-}" in
  "") ;;
  --check)
    mode="check"
    shift
    ;;
  -h|--help)
    usage
    exit 0
    ;;
  *)
    usage >&2
    die "unknown argument: $1"
    ;;
esac

[[ $# -eq 0 ]] || die "too many arguments"

repo_root=$(git rev-parse --show-toplevel 2>/dev/null) || die "run this script inside the casebook repository"
cd "$repo_root"

upstream_source="$upstream_url"
if [[ "$upstream_url" != *://* && -d "$upstream_url/.git" ]]; then
  upstream_source=$(cd "$upstream_url" && pwd -P)
fi

[[ -x "$(git --exec-path)/git-subtree" ]] || die "git-subtree is not installed"
[[ -d "$target_prefix" ]] || die "missing target directory: $target_prefix"
git ls-files -- "$target_prefix" | grep -q . || die "target directory is not tracked: $target_prefix"

if [[ "$mode" == "sync" ]]; then
  [[ -z "$(git status --porcelain=v1 --untracked-files=normal)" ]] || die "the working tree must be clean before syncing"
  [[ -z "$(git symbolic-ref -q --short HEAD)" ]] && die "sync from a branch, not a detached HEAD"
fi

baseline_commit=$(git log HEAD -1 --format='%H' --fixed-strings \
  --grep="git-subtree-dir: $target_prefix")
[[ -n "$baseline_commit" ]] || die "no subtree baseline found for $target_prefix"

baseline_body=$(git show -s --format='%B' "$baseline_commit")
recorded_split=$(printf '%s\n' "$baseline_body" | awk '$1 == "git-subtree-split:" { print $2; exit }')
case "$recorded_split" in
  ""|*[!0-9a-fA-F]*) die "invalid subtree split recorded in $baseline_commit" ;;
esac

git_dir=$(git rev-parse --absolute-git-dir)
cache_dir="$git_dir/moi-prototype-sync-cache"

if [[ ! -d "$cache_dir/.git" ]]; then
  printf 'Creating the local upstream cache…\n'
  git clone --quiet --filter=blob:none --no-checkout --single-branch --branch "$upstream_branch" \
    "$upstream_source" "$cache_dir"
  git -C "$cache_dir" sparse-checkout init --cone
  git -C "$cache_dir" sparse-checkout set "$upstream_prefix"
  git -C "$cache_dir" checkout --quiet --detach "origin/$upstream_branch"
else
  git -C "$cache_dir" remote set-url origin "$upstream_source"
  printf 'Refreshing the local upstream cache…\n'
  git -C "$cache_dir" fetch --quiet --prune origin "$upstream_branch"
  git -C "$cache_dir" sparse-checkout init --cone
  git -C "$cache_dir" sparse-checkout set "$upstream_prefix"
  git -C "$cache_dir" checkout --quiet --detach FETCH_HEAD
fi

upstream_head=$(git -C "$cache_dir" rev-parse HEAD)
latest_split=$(git -C "$cache_dir" subtree split -q --prefix="$upstream_prefix" HEAD)
case "$latest_split" in
  ""|*[!0-9a-fA-F]*) die "failed to create the upstream html split" ;;
esac
git -C "$cache_dir" update-ref refs/heads/moi-html-split "$latest_split"

printf 'Upstream main:  %s\n' "$upstream_head"
printf 'Recorded split: %s\n' "$recorded_split"
printf 'Latest split:   %s\n' "$latest_split"

if [[ "$recorded_split" == "$latest_split" ]]; then
  printf 'Already up to date.\n'
  exit 0
fi

if ! git -C "$cache_dir" cat-file -e "$recorded_split^{commit}" 2>/dev/null; then
  die "the recorded split is absent from current upstream history; rebaseline manually"
fi
if ! git -C "$cache_dir" merge-base --is-ancestor "$recorded_split" "$latest_split"; then
  die "upstream split history diverged; refusing an automatic sync"
fi

printf '\nChanges available:\n'
git -C "$cache_dir" diff --stat "$recorded_split" "$latest_split" --

if [[ "$mode" == "check" ]]; then
  printf '\nUpdate available.\n'
  exit 1
fi

short_head=$(printf '%s' "$upstream_head" | cut -c1-12)
if ! git subtree pull --prefix="$target_prefix" "$cache_dir" moi-html-split --squash \
  -m "chore: sync moi prototype html ($short_head)"; then
  printf '\nSync stopped because conflicts need to be resolved.\n' >&2
  printf 'Resolve them, run git add on the resolved files, then run git commit.\n' >&2
  printf 'To abandon this sync, run git merge --abort.\n' >&2
  exit 2
fi

if ! ./scripts/check-prototype-public-data.sh; then
  printf '\nThe upstream merge completed, but public-data review failed.\n' >&2
  printf 'Sanitize the reported values in a follow-up commit before pushing.\n' >&2
  exit 3
fi

printf '\nSync complete. Review the commit and test the prototype before pushing.\n'
