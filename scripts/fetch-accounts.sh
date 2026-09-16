#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'USAGE'
Usage: scripts/fetch-accounts.sh [ACCESS_URL]

Fetches {ACCESS_URL}/accounts with useful query options.
The access URL may be passed as an argument or via SIMPLEFIN_ACCESS_URL.

Environment options:
  VERSION=2                 Protocol version, defaults to 2
  START_DATE=1704067200     Include transactions on/after this Unix timestamp
  END_DATE=1706745600       Include transactions before this Unix timestamp
  PENDING=1                 Include pending transactions if supported
  BALANCES_ONLY=1           Fetch balances without transactions
  ACCOUNT_IDS='a1,a2'       Comma-separated account IDs; repeated account= params

Example:
  BALANCES_ONLY=1 scripts/fetch-accounts.sh "$SIMPLEFIN_ACCESS_URL"
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
args=(--fail --silent --show-error --location --get "$base/accounts")
args+=(--data-urlencode "version=${VERSION:-2}")

if [[ -n "${START_DATE:-}" ]]; then
  args+=(--data-urlencode "start-date=${START_DATE}")
fi
if [[ -n "${END_DATE:-}" ]]; then
  args+=(--data-urlencode "end-date=${END_DATE}")
fi
if [[ "${PENDING:-}" == "1" || "${PENDING:-}" == "true" ]]; then
  args+=(--data-urlencode "pending=1")
fi
if [[ "${BALANCES_ONLY:-}" == "1" || "${BALANCES_ONLY:-}" == "true" ]]; then
  args+=(--data-urlencode "balances-only=1")
fi
if [[ -n "${ACCOUNT_IDS:-}" ]]; then
  IFS=',' read -r -a account_ids <<< "$ACCOUNT_IDS"
  for account_id in "${account_ids[@]}"; do
    [[ -z "$account_id" ]] && continue
    args+=(--data-urlencode "account=${account_id}")
  done
fi

curl "${args[@]}"
printf '\n'
