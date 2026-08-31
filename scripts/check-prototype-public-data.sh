#!/usr/bin/env bash
set -euo pipefail

readonly target_prefix="product/moi-platform-prototype"
readonly known_private_names='NESR|Intelie|外研社|安利|纽崔莱|有临|Topcast|武汉新芯|芯联|芯导|CDG|方佳俊|陈工|王立华|李雪|李运维|周强|吴测试|王运维|刘运营|黄财务|林渠道'
readonly credential_shapes='(AKIA|ASIA)[A-Z0-9]{12,}|sk-[A-Za-z0-9_-]{16,}|moi_sk_[A-Za-z0-9_-]{12,}|ghp_[A-Za-z0-9]{16,}|github_pat_[A-Za-z0-9_]+'

repo_root=$(git rev-parse --show-toplevel 2>/dev/null) || {
  printf 'error: run this script inside the casebook repository\n' >&2
  exit 2
}
cd "$repo_root"

failed=0
scan_paths=(
  "$target_prefix"
  ":(exclude)$target_prefix/images/**"
  ":(exclude)$target_prefix/app-dev/assets/media/**"
)

if git grep -n -I -E "$known_private_names" -- "${scan_paths[@]}"; then
  printf '\nerror: known private customer or person names were found.\n' >&2
  failed=1
fi
if git grep -n -I -E 'matrixorigin\.(cn|com)' -- "${scan_paths[@]}"; then
  printf '\nerror: internal MatrixOrigin domains were found.\n' >&2
  failed=1
fi
if git grep -n -I -E "$credential_shapes" -- "${scan_paths[@]}"; then
  printf '\nerror: values shaped like unmasked credentials were found.\n' >&2
  failed=1
fi

if [[ "$failed" -ne 0 ]]; then
  printf 'Replace matches with public demo names or visibly masked placeholders.\n' >&2
  exit 1
fi

printf 'Prototype public-data check passed.\n'
