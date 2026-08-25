#!/usr/bin/env python3
"""Reclaim temporary UC control-api-keys so kb-product-matrix can issue a PAT.

Local-only helper for the knowledge-base product matrix gate.

UC enforces control_api_key_max_active (typically 5). Expired keys stay
status=active until revoked, so historical acceptance PATs block issue_pat
and the matrix never reaches M1–M11.

This script:
  - logs in as the seed admin
  - lists control-api-keys
  - revokes only temporary acceptance names (prefix match)
  - never revokes keep-list names (e.g. AI Studio Runtime)

Env:
  UC_BASE_URL, SEED_EMAIL, SEED_PASSWORD  (required for mutate)
  UC_PAT_PYDIR / PYTHONPATH             (moi-backend/api-tester for utils.uc_pat)
  MATRIX_PAT_RESERVE                    (free slots to ensure; default 1)
  MATRIX_PAT_TEMP_PREFIXES              (comma-separated; optional override)
  MATRIX_PAT_KEEP_PREFIXES              (comma-separated; optional override)

Usage:
  python reclaim_temp_pats.py              # free slots until reserve met
  python reclaim_temp_pats.py --list       # print active keys only
  python reclaim_temp_pats.py --force-all-temp  # revoke all matching temp keys
"""
from __future__ import annotations

import argparse
import json
import os
import sys
import uuid
from datetime import datetime, timezone
from typing import Any

import requests

# Prefer UC_PAT_PYDIR so the same path as issue_pat_once works.
_pydir = os.environ.get("UC_PAT_PYDIR", "").strip()
if _pydir:
    sys.path.insert(0, _pydir)

from utils.uc_pat import (  # noqa: E402
    UCPATError,
    _envelope_data,
    _required_header,
    _strong_key_etag,
)

_COLLECTION_ETAG_HEADER = "X-Control-Key-Collection-If-Match"

# Historical / matrix / hand-accept temporary prefixes. Keep in sync with SKILL.md.
_DEFAULT_TEMP_PREFIXES = (
    "kb-matrix-acc-",
    "kb-lineage-acc-",
    "kb-accept-",
    "kb-dbchk-",
    "kb-reg-",
    "moi-tester-",
    "moi-tester-agent-",
)

_DEFAULT_KEEP_PREFIXES = (
    "AI Studio Runtime",
    "ai-studio-runtime",
)


def _parse_prefixes(env_name: str, defaults: tuple[str, ...]) -> tuple[str, ...]:
    raw = os.environ.get(env_name, "").strip()
    if not raw:
        return defaults
    parts = [p.strip() for p in raw.split(",") if p.strip()]
    return tuple(parts) if parts else defaults


def _is_temp(name: str, temp_prefixes: tuple[str, ...], keep_prefixes: tuple[str, ...]) -> bool:
    if not name:
        return False
    for kp in keep_prefixes:
        if name == kp or name.startswith(kp):
            return False
    for tp in temp_prefixes:
        if name.startswith(tp):
            return True
    return False


def _login(client: requests.Session, uc: str, email: str, password: str) -> str:
    login = client.post(
        f"{uc}/api/v1/uc/sessions/login",
        json={"email": email, "password": password, "app": "ai", "return_to": "/"},
        timeout=60,
    )
    data = _envelope_data(login, "UC login")
    csrf = data.get("csrf_token")
    if not isinstance(csrf, str) or not csrf:
        raise UCPATError("UC login response missing csrf_token")
    return csrf


def _list_keys(client: requests.Session, uc: str) -> tuple[list[dict[str, Any]], str]:
    lr = client.get(f"{uc}/api/v1/uc/me/control-api-keys", timeout=60)
    data = _envelope_data(lr, "list control-api-keys")
    coll = _required_header(lr, _COLLECTION_ETAG_HEADER, "list control-api-keys")
    items = data.get("items")
    if not isinstance(items, list):
        raise UCPATError("control-api-keys list missing items array")
    out: list[dict[str, Any]] = []
    for it in items:
        if isinstance(it, dict):
            out.append(it)
    return out, coll


def _capabilities_max_active(client: requests.Session, uc: str) -> int | None:
    try:
        cr = client.get(f"{uc}/api/v1/uc/me/service-accounts/capabilities", timeout=60)
        data = _envelope_data(cr, "capabilities")
        max_active = data.get("control_api_key_max_active")
        if isinstance(max_active, int) and max_active > 0:
            return max_active
    except UCPATError:
        return None
    return None


def _key_etag(client: requests.Session, uc: str, key_id: str) -> str:
    gr = client.get(f"{uc}/api/v1/uc/me/control-api-keys/{key_id}", timeout=60)
    _envelope_data(gr, f"get key {key_id}")
    etag = gr.headers.get("ETag")
    if not isinstance(etag, str) or not etag.strip():
        raise UCPATError(f"get key {key_id} missing ETag")
    return _strong_key_etag(etag, f"get key {key_id}")


def _revoke(
    client: requests.Session,
    uc: str,
    csrf: str,
    key_id: str,
    key_etag: str,
    collection_etag: str,
) -> None:
    dr = client.delete(
        f"{uc}/api/v1/uc/me/control-api-keys/{key_id}",
        headers={
            "X-CSRF-Token": csrf,
            "If-Match": key_etag,
            _COLLECTION_ETAG_HEADER: collection_etag,
            "Idempotency-Key": str(uuid.uuid4()),
        },
        timeout=60,
    )
    _envelope_data(dr, f"revoke {key_id}")


def _active(items: list[dict[str, Any]]) -> list[dict[str, Any]]:
    return [it for it in items if it.get("status") == "active"]


def _sort_reclaim_candidates(items: list[dict[str, Any]]) -> list[dict[str, Any]]:
    """Prefer expired, then oldest created_at."""

    def sort_key(it: dict[str, Any]) -> tuple[int, str]:
        exp = it.get("expires_at") or ""
        created = it.get("created_at") or ""
        expired = 0
        if isinstance(exp, str) and exp:
            try:
                # handle trailing Z
                dt = datetime.fromisoformat(exp.replace("Z", "+00:00"))
                if dt <= datetime.now(timezone.utc):
                    expired = 0  # expired first
                else:
                    expired = 1
            except ValueError:
                expired = 1
        else:
            expired = 1
        return (expired, created)

    return sorted(items, key=sort_key)


def main() -> int:
    parser = argparse.ArgumentParser(description="Reclaim temporary UC PATs for kb-product-matrix")
    parser.add_argument("--list", action="store_true", help="List keys only; do not revoke")
    parser.add_argument(
        "--force-all-temp",
        action="store_true",
        help="Revoke every active temp-prefix key (ignore reserve)",
    )
    parser.add_argument(
        "--reserve",
        type=int,
        default=None,
        help="Ensure at least N free slots (default MATRIX_PAT_RESERVE or 1)",
    )
    args = parser.parse_args()

    uc = os.environ.get("UC_BASE_URL", "").rstrip("/")
    email = os.environ.get("SEED_EMAIL", "")
    password = os.environ.get("SEED_PASSWORD", "")
    if not uc or not email or not password:
        print(
            "error: need UC_BASE_URL, SEED_EMAIL, SEED_PASSWORD",
            file=sys.stderr,
        )
        return 2

    temp_prefixes = _parse_prefixes("MATRIX_PAT_TEMP_PREFIXES", _DEFAULT_TEMP_PREFIXES)
    keep_prefixes = _parse_prefixes("MATRIX_PAT_KEEP_PREFIXES", _DEFAULT_KEEP_PREFIXES)
    reserve = args.reserve
    if reserve is None:
        try:
            reserve = int(os.environ.get("MATRIX_PAT_RESERVE", "1"))
        except ValueError:
            reserve = 1
    if reserve < 0:
        reserve = 0

    client = requests.Session()
    try:
        csrf = _login(client, uc, email, password)
        items, coll = _list_keys(client, uc)
        max_active = _capabilities_max_active(client, uc)
        active = _active(items)

        summary = {
            "uc_base_url": uc,
            "email": email,
            "active": len(active),
            "max_active": max_active,
            "reserve": reserve,
            "temp_prefixes": list(temp_prefixes),
            "keys": [
                {
                    "id": it.get("id"),
                    "name": it.get("name"),
                    "status": it.get("status"),
                    "expires_at": it.get("expires_at"),
                    "temp": _is_temp(str(it.get("name") or ""), temp_prefixes, keep_prefixes),
                }
                for it in items
            ],
        }
        print(json.dumps({"phase": "list", **summary}, ensure_ascii=False, indent=2))

        if args.list:
            free = None if max_active is None else max(0, max_active - len(active))
            if free is not None and free < reserve:
                print(
                    f"[reclaim-temp-pats] WARN free_slots={free} < reserve={reserve} "
                    f"(active={len(active)}, max={max_active})",
                    file=sys.stderr,
                )
                return 1
            return 0

        # UC default max is 5; capabilities may omit the field on older builds.
        effective_max = max_active if max_active is not None else 5
        free = max(0, effective_max - len(active))

        candidates = [
            it
            for it in _sort_reclaim_candidates(active)
            if _is_temp(str(it.get("name") or ""), temp_prefixes, keep_prefixes)
            and isinstance(it.get("id"), str)
            and it["id"]
        ]

        if args.force_all_temp:
            targets = candidates
            to_free = len(targets)
        elif free >= reserve:
            print(
                f"[reclaim-temp-pats] ok free_slots={free} >= reserve={reserve}; no revoke",
                file=sys.stderr,
            )
            return 0
        else:
            to_free = reserve - free
            targets = candidates[:to_free]

        if to_free > 0 and not targets:
            print(
                "[reclaim-temp-pats] ERROR: need free PAT slots but no temporary keys match "
                f"prefixes {list(temp_prefixes)}. Active keys:\n"
                + "\n".join(
                    f"  - {it.get('name')} id={it.get('id')} expires={it.get('expires_at')}"
                    for it in active
                )
                + "\nRevoke non-temp keys manually in UC, or extend MATRIX_PAT_TEMP_PREFIXES.",
                file=sys.stderr,
            )
            return 1

        revoked: list[dict[str, Any]] = []
        for it in targets:
            kid = str(it["id"])
            name = str(it.get("name") or "")
            items, coll = _list_keys(client, uc)
            # refresh etag after each mutation
            try:
                key_etag = _key_etag(client, uc, kid)
                _revoke(client, uc, csrf, kid, key_etag, coll)
            except UCPATError as exc:
                print(f"[reclaim-temp-pats] FAIL revoke {name}: {exc}", file=sys.stderr)
                return 1
            revoked.append({"id": kid, "name": name})
            print(f"[reclaim-temp-pats] revoked {name} ({kid})", file=sys.stderr)

        items, _ = _list_keys(client, uc)
        active_after = _active(items)
        free_after = None if max_active is None else max(0, max_active - len(active_after))
        result = {
            "phase": "done",
            "revoked": revoked,
            "active_before": len(active),
            "active_after": len(active_after),
            "max_active": max_active,
            "free_after": free_after,
            "reserve": reserve,
        }
        print(json.dumps(result, ensure_ascii=False, indent=2))

        if free_after is not None and free_after < reserve and not args.force_all_temp:
            print(
                f"[reclaim-temp-pats] ERROR still free_slots={free_after} < reserve={reserve}",
                file=sys.stderr,
            )
            return 1
        print(
            f"[reclaim-temp-pats] ok active={len(active_after)} free={free_after} reserve={reserve}",
            file=sys.stderr,
        )
        return 0
    finally:
        client.close()


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except UCPATError as exc:
        print(f"error: {exc}", file=sys.stderr)
        raise SystemExit(1) from None
