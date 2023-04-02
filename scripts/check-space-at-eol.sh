#!/usr/bin/env bash

set -euo pipefail

cd -P -- "$(dirname -- "$0")/.."

n=0
while read -r file; do
    if grep -E -q " +$" "$file"; then
        echo "space_at_eol: $file"
        n=$(( n + 1 ))
    fi
done < <(git ls-files | grep -v -e LICENSES -e '\.png$')

exit $n
