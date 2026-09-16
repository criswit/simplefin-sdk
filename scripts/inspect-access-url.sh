#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'USAGE'
Usage: scripts/inspect-access-url.sh [ACCESS_URL]

Parses a SimpleFIN access URL and prints redacted components.
You can pass the URL as an argument or via SIMPLEFIN_ACCESS_URL.
USAGE
}

access_url="${1:-${SIMPLEFIN_ACCESS_URL:-}}"
if [[ -z "$access_url" ]]; then
  usage
  exit 2
fi

ACCESS_URL="$access_url" python3 - <<'PY'
import os
import sys
from urllib.parse import urlsplit, urlunsplit, unquote

raw = os.environ["ACCESS_URL"]
u = urlsplit(raw)
if u.scheme != "https":
    sys.exit("error: access URL must use https")
if not u.username or u.password is None or u.password == "":
    sys.exit("error: access URL must include username and password")

netloc = u.hostname or ""
if u.port:
    netloc += f":{u.port}"
base = urlunsplit((u.scheme, netloc, u.path.rstrip("/"), "", ""))
redacted = urlunsplit((u.scheme, f"{unquote(u.username)}:***@{netloc}", u.path.rstrip("/"), "", ""))

print(f"scheme={u.scheme}")
print(f"base_url={base}")
print(f"username={unquote(u.username)}")
print("password=***")
print(f"redacted_access_url={redacted}")
print(f"info_endpoint={base}/info")
print(f"accounts_endpoint={base}/accounts")
PY
