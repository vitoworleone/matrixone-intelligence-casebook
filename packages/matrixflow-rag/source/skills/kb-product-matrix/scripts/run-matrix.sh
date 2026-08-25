#!/usr/bin/env bash
# Run Knowledge Base Product SDK matrix against a live local-deploy stack.
# Exit 0 only when status=MATRIX_PASSED and all checks pass.
#
# Artifacts under MATRIX_OUT and the local runner binary are gitignored.
# After the run finishes, this script asks whether to delete those artifacts
# (unless MATRIX_CLEANUP=yes|no). Agents should surface the prompt to the user.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILL_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
ROOT="$(cd "$SKILL_DIR/../.." && pwd)"
RUNNER_DIR="$SKILL_DIR/runner"
RUNNER_BIN="$RUNNER_DIR/kb-product-matrix"
REVOKE_SCRIPT="$RUNNER_DIR/revoke_pat_once.py"

if [[ ! -d "$ROOT/sdk/go-sdk" ]]; then
  echo "error: cannot find repo root with sdk/go-sdk (resolved ROOT=$ROOT)" >&2
  exit 2
fi

# Optional: load UC local-deploy env for ports/profile.
if [[ -f "$ROOT/.runtime/local-deploy.uc.env" ]]; then
  # shellcheck disable=SC1091
  set -a
  # shellcheck disable=SC1090
  source "$ROOT/.runtime/local-deploy.uc.env"
  set +a
fi

export WORKTREE_ROOT="${WORKTREE_ROOT:-$ROOT}"
export MATRIXFLOW_ROOT="${MATRIXFLOW_ROOT:-$ROOT}"
export AISTUDIO_PORT="${AISTUDIO_PORT:-19050}"
export UC_PORT="${UC_PORT:-19080}"
export AISTUDIO_PUBLIC_URL="${AISTUDIO_PUBLIC_URL:-http://localhost:18000}"
export ACCOUNT_PUBLIC_ORIGIN="${ACCOUNT_PUBLIC_ORIGIN:-http://127.0.0.1:8100}"
# Designated output tree only; cleanup must never rm outside this prefix.
MATRIX_OUT_BASE="$ROOT/output/kb-product-matrix"
export MATRIX_OUT="${MATRIX_OUT:-$MATRIX_OUT_BASE}"

# Resolve to absolute path. Non-existent paths resolve via parent (parent must exist).
resolve_abs_path() {
  local path="$1"
  local dir base
  if [[ -z "$path" ]]; then
    return 1
  fi
  if [[ -d "$path" ]]; then
    (cd "$path" && pwd -P)
    return 0
  fi
  if [[ -e "$path" ]]; then
    dir="$(cd "$(dirname "$path")" && pwd -P)" || return 1
    base="$(basename "$path")"
    printf '%s/%s\n' "$dir" "$base"
    return 0
  fi
  dir="$(cd "$(dirname "$path")" 2>/dev/null && pwd -P)" || return 1
  base="$(basename "$path")"
  printf '%s/%s\n' "$dir" "$base"
}

# MATRIX_OUT must be the designated base or a child of it (never /, repo root, or outside).
assert_safe_matrix_out() {
  local target allowed root_abs
  if [[ -z "${MATRIX_OUT:-}" ]]; then
    echo "error: MATRIX_OUT is empty; refusing cleanup/write" >&2
    return 1
  fi
  # Ensure base parent exists so resolve works for first-time runs.
  mkdir -p "$(dirname "$MATRIX_OUT_BASE")"
  mkdir -p "$MATRIX_OUT_BASE"
  target="$(resolve_abs_path "$MATRIX_OUT")" || {
    echo "error: cannot resolve MATRIX_OUT=$MATRIX_OUT" >&2
    return 1
  }
  allowed="$(resolve_abs_path "$MATRIX_OUT_BASE")" || {
    echo "error: cannot resolve MATRIX_OUT_BASE=$MATRIX_OUT_BASE" >&2
    return 1
  }
  root_abs="$(resolve_abs_path "$ROOT")" || {
    echo "error: cannot resolve ROOT=$ROOT" >&2
    return 1
  }
  if [[ "$target" == "/" || "$target" == "$root_abs" ]]; then
    echo "error: MATRIX_OUT=$target is not allowed (root or repository root)" >&2
    return 1
  fi
  if [[ "$target" != "$allowed" && "$target" != "$allowed"/* ]]; then
    echo "error: MATRIX_OUT=$target must be under $allowed" >&2
    return 1
  fi
  MATRIX_OUT="$target"
  export MATRIX_OUT
}

cleanup_matrix_artifacts() {
  local removed=0
  assert_safe_matrix_out || return 1
  if [[ -d "$MATRIX_OUT" ]]; then
    echo "[kb-product-matrix] removing MATRIX_OUT=$MATRIX_OUT" >&2
    rm -rf -- "$MATRIX_OUT"
    removed=1
  fi
  # Only delete known skill-local binaries under RUNNER_DIR (not caller-controlled paths).
  if [[ -f "$RUNNER_BIN" && "$RUNNER_BIN" == "$RUNNER_DIR"/* ]]; then
    rm -f -- "$RUNNER_BIN"
    removed=1
  fi
  if [[ -f "$REVOKE_SCRIPT" && "$REVOKE_SCRIPT" == "$RUNNER_DIR"/* ]]; then
    rm -f -- "$REVOKE_SCRIPT"
    removed=1
  fi
  if [[ "$removed" -eq 1 ]]; then
    echo "[kb-product-matrix] local artifacts cleaned" >&2
  else
    echo "[kb-product-matrix] no local artifacts to clean" >&2
  fi
}

# --cleanup-only: remove prior run artifacts without re-running the matrix.
if [[ "${1:-}" == "--cleanup-only" ]]; then
  cleanup_matrix_artifacts
  exit 0
fi

PROFILE="${LOCAL_DEPLOY_PROFILE:-${UC_PROFILE:-kb_catalog_lineage_acceptance}}"
export LOCAL_DEPLOY_PROFILE="$PROFILE"
export SEED_EMAIL="${SEED_EMAIL:-local-admin+${PROFILE}@matrixflow.local}"
export SEED_PASSWORD="${SEED_PASSWORD:-Admin@1234}"

# Python for UC PAT issue (needs requests).
if [[ -z "${PYTHON_BIN:-}" ]]; then
  for cand in \
    "$ROOT/.runtime/kb-catalog-lineage-acceptance-runner/venv/bin/python" \
    "$ROOT/.runtime/kb-product-matrix-runner/venv/bin/python" \
    "$RUNNER_DIR/venv/bin/python"; do
    if [[ -x "$cand" ]]; then
      PYTHON_BIN="$cand"
      break
    fi
  done
fi
export PYTHON_BIN="${PYTHON_BIN:-python3}"

if ! "$PYTHON_BIN" -c 'import requests' 2>/dev/null; then
  echo "error: PYTHON_BIN=$PYTHON_BIN missing 'requests'. Create a venv with requests or set PYTHON_BIN." >&2
  echo "  example: python3 -m venv $RUNNER_DIR/venv && $RUNNER_DIR/venv/bin/pip install requests" >&2
  exit 2
fi

# Shared env for PAT helpers (issue / reclaim / revoke).
export UC_BASE_URL="${UC_BASE_URL:-http://127.0.0.1:${UC_PORT}}"
export PRODUCT_BASE_URL="${PRODUCT_BASE_URL:-${AISTUDIO_PUBLIC_URL%/}/newmoi}"
export UC_PAT_PYDIR="${UC_PAT_PYDIR:-$ROOT/moi-backend/api-tester}"
export PYTHONPATH="${PYTHONPATH:-}${PYTHONPATH:+:}$UC_PAT_PYDIR"
export MATRIX_PAT_RESERVE="${MATRIX_PAT_RESERVE:-1}"
# MATRIX_PAT_RECLAIM: auto (default) | yes|force | no|skip | list
export MATRIX_PAT_RECLAIM="${MATRIX_PAT_RECLAIM:-auto}"

RECLAIM_SCRIPT="$RUNNER_DIR/reclaim_temp_pats.py"

run_pat_reclaim() {
  local mode mode_lc
  mode="${MATRIX_PAT_RECLAIM:-auto}"
  mode_lc="$(printf '%s' "$mode" | tr '[:upper:]' '[:lower:]')"
  case "$mode_lc" in
    no|false|0|n|skip|off)
      echo "[kb-product-matrix] MATRIX_PAT_RECLAIM=$mode — skip UC PAT reclaim preflight" >&2
      return 0
      ;;
  esac
  if [[ ! -f "$RECLAIM_SCRIPT" ]]; then
    echo "error: missing $RECLAIM_SCRIPT (required to free UC PAT slots before matrix)" >&2
    return 2
  fi
  echo "[kb-product-matrix] PAT preflight: reclaim temp keys (mode=$mode_lc reserve=$MATRIX_PAT_RESERVE uc=$UC_BASE_URL)" >&2
  case "$mode_lc" in
    list)
      "$PYTHON_BIN" "$RECLAIM_SCRIPT" --list
      return $?
      ;;
    yes|true|1|y|force|all)
      "$PYTHON_BIN" "$RECLAIM_SCRIPT" --force-all-temp
      return $?
      ;;
    auto|*)
      # Free slots until reserve met; only temp acceptance prefixes.
      "$PYTHON_BIN" "$RECLAIM_SCRIPT" --reserve "$MATRIX_PAT_RESERVE"
      return $?
      ;;
  esac
}

# --reclaim-pats-only: free UC temporary PATs without running the matrix.
if [[ "${1:-}" == "--reclaim-pats-only" ]]; then
  run_pat_reclaim
  exit $?
fi

# --list-pats-only: show active control-api-keys for seed admin.
if [[ "${1:-}" == "--list-pats-only" ]]; then
  MATRIX_PAT_RECLAIM=list run_pat_reclaim
  exit $?
fi

# Preflight: backend must answer (401 unauthenticated is healthy).
BE_CODE="$(curl --noproxy '*' -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:${AISTUDIO_PORT}/newmoi/auth/me" || true)"
FE_CODE="$(curl --noproxy '*' -s -o /dev/null -w '%{http_code}' "${AISTUDIO_PUBLIC_URL}/newmoi/auth/me" || true)"
UC_CODE="$(curl --noproxy '*' -s -o /dev/null -w '%{http_code}' "${UC_BASE_URL}/api/v1/uc/healthz" 2>/dev/null || curl --noproxy '*' -s -o /dev/null -w '%{http_code}' "${UC_BASE_URL}/healthz" 2>/dev/null || true)"
if [[ "$BE_CODE" != "401" && "$BE_CODE" != "200" ]]; then
  echo "error: moi-backend not healthy at :${AISTUDIO_PORT} (HTTP $BE_CODE). Start/restart local-deploy first." >&2
  exit 2
fi
if [[ "$FE_CODE" != "401" && "$FE_CODE" != "200" && "$FE_CODE" != "302" ]]; then
  echo "error: frontend proxy not healthy at ${AISTUDIO_PUBLIC_URL} (HTTP $FE_CODE). PAT OIDC needs FE /newmoi." >&2
  exit 2
fi

echo "[kb-product-matrix] root=$ROOT profile=$PROFILE backend=:${AISTUDIO_PORT} frontend=$AISTUDIO_PUBLIC_URL" >&2
echo "[kb-product-matrix] seed=$SEED_EMAIL python=$PYTHON_BIN out=$MATRIX_OUT" >&2

# PAT slot preflight: expired/leftover temp keys still count toward max_active (usually 5).
# Without this, issue_pat fails before any M1–M11 product check runs.
if ! run_pat_reclaim; then
  echo "error: UC PAT reclaim preflight failed." >&2
  echo "  Symptom: issue_pat → 'UC active PAT 已达到环境上限: active=5, max=5'" >&2
  echo "  Fix:     bash skills/kb-product-matrix/scripts/run-matrix.sh --reclaim-pats-only" >&2
  echo "  List:    bash skills/kb-product-matrix/scripts/run-matrix.sh --list-pats-only" >&2
  echo "  Force:   MATRIX_PAT_RECLAIM=force bash skills/kb-product-matrix/scripts/run-matrix.sh --reclaim-pats-only" >&2
  echo "  Manual:  revoke kb-matrix-acc-* / kb-accept-* / kb-lineage-acc-* / kb-dbchk-* in UC; keep AI Studio Runtime." >&2
  exit 2
fi

assert_safe_matrix_out
mkdir -p "$MATRIX_OUT"
cd "$RUNNER_DIR"
go mod tidy >/dev/null
go build -o kb-product-matrix .

export ACCEPTANCE_SUFFIX="${ACCEPTANCE_SUFFIX:-matrix-$(date +%Y%m%d-%H%M%S)}"

matrix_exit=0
set +e
./kb-product-matrix | tee "$MATRIX_OUT/matrix-summary.json"
matrix_exit=${PIPESTATUS[0]}
set -e

# If matrix died at issue_pat with PAT ceiling, print actionable recovery once.
if [[ "$matrix_exit" -ne 0 ]]; then
  if grep -q 'UC active PAT 已达到环境上限\|active PAT' "$MATRIX_OUT/matrix-summary.json" 2>/dev/null \
    || grep -q 'UC active PAT 已达到环境上限\|active PAT' "$MATRIX_OUT/matrix-report.json" 2>/dev/null; then
    echo "[kb-product-matrix] HINT: PAT ceiling — run --reclaim-pats-only then re-run matrix (not a KB product failure)." >&2
  fi
fi

echo "[kb-product-matrix] full report: $MATRIX_OUT/matrix-report.json" >&2
if [[ "$matrix_exit" -eq 0 ]]; then
  echo "[kb-product-matrix] status=MATRIX_PASSED exit=0" >&2
else
  echo "[kb-product-matrix] status=FAILED exit=$matrix_exit (see matrix-report.json)" >&2
fi

# --- Artifact inventory (gitignored; optional cleanup) ---
# MATRIX_CLEANUP:
#   yes|true|1  — delete without prompt
#   no|false|0  — keep without prompt
#   ask|unset   — prompt when stdin is a TTY; otherwise print agent-facing question and keep
list_matrix_artifacts() {
  echo "[kb-product-matrix] local artifacts (not git-tracked):" >&2
  if [[ -d "$MATRIX_OUT" ]]; then
    # shellcheck disable=SC2012
    ls -la "$MATRIX_OUT" 2>/dev/null | sed 's/^/[kb-product-matrix]   /' >&2 || true
    echo "[kb-product-matrix]   dir: $MATRIX_OUT" >&2
  else
    echo "[kb-product-matrix]   (no MATRIX_OUT dir)" >&2
  fi
  [[ -f "$RUNNER_BIN" ]] && echo "[kb-product-matrix]   bin: $RUNNER_BIN" >&2
  [[ -f "$REVOKE_SCRIPT" ]] && echo "[kb-product-matrix]   tmp: $REVOKE_SCRIPT" >&2
}

maybe_cleanup_after_run() {
  list_matrix_artifacts

  local mode="${MATRIX_CLEANUP:-ask}"
  mode="$(printf '%s' "$mode" | tr '[:upper:]' '[:lower:]')"
  case "$mode" in
    yes|true|1|y)
      cleanup_matrix_artifacts
      return 0
      ;;
    no|false|0|n)
      echo "[kb-product-matrix] MATRIX_CLEANUP=$mode — keeping artifacts" >&2
      return 0
      ;;
  esac

  if [[ -t 0 ]]; then
    echo >&2
    echo "[kb-product-matrix] 是否清理本次执行产物？(y/N)" >&2
    echo "[kb-product-matrix]  将删除: $MATRIX_OUT" >&2
    echo "[kb-product-matrix]          $RUNNER_BIN (if present)" >&2
    echo "[kb-product-matrix]          $REVOKE_SCRIPT (if present)" >&2
    local ans=""
    # read from /dev/tty so pipeline stdin does not steal the answer
    if [[ -r /dev/tty ]]; then
      read -r -p "[kb-product-matrix] cleanup artifacts? [y/N] " ans </dev/tty || true
    else
      read -r -p "[kb-product-matrix] cleanup artifacts? [y/N] " ans || true
    fi
    case "${ans:-}" in
      y|Y|yes|YES)
        cleanup_matrix_artifacts
        ;;
      *)
        echo "[kb-product-matrix] keeping artifacts (user declined or empty)" >&2
        ;;
    esac
    return 0
  fi

  # Non-interactive (agent / CI): do not delete; instruct caller to ask the user.
  cat >&2 <<EOF
[kb-product-matrix] CLEANUP_PROMPT
[kb-product-matrix] 矩阵已结束 (exit=$matrix_exit)。产物未自动删除（非交互模式）。
[kb-product-matrix] Agent 必须询问用户是否清理，再执行其一：
[kb-product-matrix]   清理: MATRIX_CLEANUP=yes bash skills/kb-product-matrix/scripts/run-matrix.sh --cleanup-only
[kb-product-matrix]   或:   rm -rf "$MATRIX_OUT" && rm -f "$RUNNER_BIN" "$REVOKE_SCRIPT"
[kb-product-matrix]   保留: 无需操作（目录已 gitignore: output/kb-product-matrix/）
[kb-product-matrix] CLEANUP_PROMPT_END
EOF
}

maybe_cleanup_after_run
exit "$matrix_exit"
