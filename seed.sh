#!/usr/bin/env bash
# Throwaway script to seed infraview with test plans.
# Usage: ./seed.sh
# Assumes server running at localhost:8080 with --no-auth.

set -euo pipefail

SERVER="http://localhost:8080"

# Map test dir -> namespace
namespace() {
  case "$1" in
    00-simple-resource|01-complex-resource|02-simple-create-update-delete|03-simple-dependencies) echo "basic" ;;
    04-simple-vars|06-simple-outputs)    echo "vars-outputs" ;;
    05-module-vars|07-module-outputs|08-complex-vars|10-module-deletions) echo "modules" ;;
    09-data-resources)                   echo "data" ;;
    12-create-count|13-create-foreach|14-delete-count) echo "loops" ;;
    11-deletion-edges|15-test-collapse|99|99-hard) echo "misc" ;;
    *) echo "unknown" ;;
  esac
}

cd "$(dirname "$0")"

HEADERS=()
if [ -n "${1-}" ]; then
  HEADERS+=(-H "Authorization: Bearer $1") # if you really want to seed an authorization-enabled infraview, pass write the token as first parameter
fi

for plan in test/graph/*/plan.json; do
  [ -f "$plan" ] || continue
  [ -s "$plan" ] || continue

  dir=$(basename "$(dirname "$plan")")
  name="${dir#test/graph}"
  name="${name%/plan.json}"
  ns=$(namespace "$dir")
  rev=$(sha256sum "$plan" | head -c 8)

  code=$(curl -s -o /dev/null -w '%{http_code}' -X PUT \
    --data-binary @"$plan" \
    -H 'Content-Type: application/json' "${HEADERS[@]}" \
    "${SERVER}/api/v1/graph/${ns}/${rev}?name=\"$name\"")

  echo "Ingested plan \"${dir}\" → ${ns}/${rev} (${code})"
done
