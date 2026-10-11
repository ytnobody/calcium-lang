#!/bin/bash
# check-spec-stdlib.sh - Fail CI if a core.string function is undocumented in the spec.
#
# Keeps calcium-spec.md complete: every public function declared in the stdlib
# implementation (pkg/vm/stdlib/core/string.ca) must be mentioned in the spec
# (as `string.<name>`). Additive stdlib changes require a spec update in the
# same PR.

set -e

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

SPEC="calcium-spec.md"
FAIL=0

STDLIB_FILES=$(find pkg/vm/stdlib/core -name '*.ca' 2>/dev/null || true)

# Only core.string is checked declaratively here; other core modules use a mix
# of primitives and composition. Extend the list as modules are documented.
for f in pkg/vm/stdlib/core/string.ca; do
  if [[ ! -f "$f" ]]; then
    echo "error: stdlib file $f not found"
    exit 1
  fi
  while read -r fn; do
    if [[ -z "$fn" ]]; then
      continue
    fi
    if ! grep -q "string\.$fn" "$SPEC"; then
      echo "error: core.string function 'string.$fn' is not documented in $SPEC (see '### string Namespace')"
      FAIL=1
    fi
  done < <(grep -oE '^func [a-z_]+' "$f" | sed 's/^func //')
done

if [[ "$FAIL" -ne 0 ]]; then
  echo
  echo "Spec is out of date. Add the missing core.string functions to '$SPEC'."
  exit 1
fi

echo "OK: every core.string function is documented in $SPEC"