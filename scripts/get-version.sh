#!/usr/bin/env bash
set -euo pipefail

# ensure dir
cd -P -- "$(dirname -- "$0")"

if git describe --tag | grep -E -q "^[0-9]+\.[0-9]+\.[0-9]$"; then
    git describe --tag
else
    git rev-parse --short HEAD
fi