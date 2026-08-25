"""Local-only PAT issue. Not for GitHub (.runtime/ gitignored)."""
import json, os, sys
sys.path.insert(0, os.environ["UC_PAT_PYDIR"])
from utils.uc_pat import issue_uc_pat

lease = issue_uc_pat(
    uc_base_url=os.environ["UC_BASE_URL"],
    product_base_url=os.environ["PRODUCT_BASE_URL"],  # FE /newmoi for OIDC session
    email=os.environ["SEED_EMAIL"],
    password=os.environ["SEED_PASSWORD"],
    ttl_seconds=int(os.environ.get("PAT_TTL", "7200")),
    timeout_seconds=60,
    name_prefix="kb-matrix-acc",
)
# If principal has no moi_user_id yet, first product touch with session provisions it.
# Browser session cookies are on the same client used by issue_uc_pat only during issue;
# for a brand-new profile, call FE/backend /workspaces once with that session before PAT.
# This profile already has moi_user_id; no extra step.
print(json.dumps({
    "token": lease.token,
    "key_id": lease.key_id,
    "key_etag": lease.key_etag,
    "collection_etag": lease.collection_etag,
    "csrf_token": lease.csrf_token,
    "uc_base_url": lease.uc_base_url,
}))
lease.session.close()
