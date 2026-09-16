#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'USAGE'
Usage: scripts/fetch-info.sh [ACCESS_URL]

Fetches {ACCESS_URL}/info. The access URL may be passed as an argument
or via SIMPLEFIN_ACCESS_URL.
USAGE
}

access_url="${1:-${SIMPLEFIN_ACCESS_URL:-}}"
if [[ -z "$access_url" ]]; then
  usage
  exit 2
fi

if [[ "$access_url" != https://* ]]; then
  echo "error: refusing to use non-HTTPS access URL" >&2
  exit 1
fi

base="${access_url%/}"
curl --fail --silent --show-error --location "$base/info"
printf '\n'
