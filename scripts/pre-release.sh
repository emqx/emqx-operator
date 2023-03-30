#!/usr/bin/env bash
set -euo pipefail

tag=$1

# ensure dir
cd -P -- "$(dirname -- "${BASH_SOURCE[0]}")/.."

# Default case for GUN sed, use "sed -i"
SED_REPLACE="sed -i "
case $(sed --help 2>&1) in
    *GNU*) SED_REPLACE="sed -i ";;
    *BusyBox*) SED_REPLACE="sed -i ";;
    *) SED_REPLACE="sed -i () ";;
esac

${SED_REPLACE} -r "s|^version:.*|version: ${tag}|g" deploy/charts/emqx-operator/Chart.yaml
${SED_REPLACE} -r "s|^appVersion:.*|appVersion: ${tag}|g" deploy/charts/emqx-operator/Chart.yaml
