#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'USAGE'
Usage: scripts/claim-setup-token.sh [SETUP_TOKEN]

Claims a SimpleFIN setup token and prints the resulting access URL to stdout.
You can pass the token as an argument or via SIMPLEFIN_SETUP_TOKEN.

Example:
  ACCESS_URL=$(scripts/claim-setup-token.sh "$SIMPLEFIN_SETUP_TOKEN")
USAGE
}

setup_token="${1:-${SIMPLEFIN_SETUP_TOKEN:-}}"
if [[ -z "$setup_token" ]]; then
  usage
  exit 2
fi

claim_url=$(printf '%s' "$setup_token" | base64 --decode 2>/dev/null || true)
if [[ -z "$claim_url" ]]; then
  echo "error: setup token is not valid base64" >&2
  exit 1
fi

if [[ "$claim_url" != https://* ]]; then
  echo "error: refusing to claim non-HTTPS URL: $claim_url" >&2
  exit 1
fi

curl --fail --silent --show-error \
  --request POST \
  --header 'Content-Length: 0' \
  "$claim_url"
printf '\n'
