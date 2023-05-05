#!/usr/bin/env bash

set -euo pipefail

cd -P -- "$(dirname -- "$0")/.."

n=0
while read -r file; do
    if grep -E -q " +$" "$file"; then
        lineNum=$(grep -n -E " +$" "$file" | cut -d: -f1 | tr '\n' ','  | sed 's/,$//')
        echo "have space at end of $lineNum line in $file"
        n=$(( n + 1 ))
    fi
done < <(git ls-files | grep -v -e LICENSES -e '\.png$')

exit $n
