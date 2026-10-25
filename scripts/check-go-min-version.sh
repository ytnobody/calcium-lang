#!/bin/bash
# check-go-min-version.sh - Fail CI if the running Go is older than go.mod
# requires.
#
# Usage: check-go-min-version.sh <running-go-version>
# Example: check-go-min-version.sh 1.26

set -e

CUR="$1"
if [[ -z "$CUR" ]]; then
  echo "usage: $0 <running-go-version>"
  exit 1
fi

if [[ ! -f go.mod ]]; then
  echo "error: no go.mod found in $(pwd)"
  exit 1
fi

REQ=$(awk '$1=="go"{print $2; exit}' go.mod | tr -d '"')
CUR="${CUR#go}"
REQ="${REQ#go}"

if [[ -z "$REQ" ]]; then
  echo "warning: go.mod has no 'go' directive; skipping check"
  exit 0
fi

# Compare major.minor only (1.26.0 == 1.26) using the setup-go short form.
majmin() { echo "$1" | cut -d. -f1-2; }
REQ_MM=$(majmin "$REQ")
CUR_MM=$(majmin "$CUR")

# sort -V ascending: head is the minimum. REQ must be <= CUR.
if [[ "$(printf '%s\n%s\n' "$REQ_MM" "$CUR_MM" | sort -V | head -1)" != "$REQ_MM" ]]; then
  echo "error: go.mod requires go >= $REQ_MM but CI is running go $CUR"
  echo "       add or update the matching go-version row in .github/workflows/ci.yml"
  exit 1
fi

echo "go.mod requires go >= $REQ_MM; running go $CUR: OK"