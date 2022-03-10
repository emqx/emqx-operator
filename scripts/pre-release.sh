#!/usr/bin/env bash
set -euo pipefail

tag=$1

# ensure dir
cd -P -- "$(dirname -- "${BASH_SOURCE[0]}")/.."

make kustomize
KUSTOMIZE=${PWD}/bin/kustomize
pushd config/manager && ${KUSTOMIZE} edit set image controller="emqx/emqx-operator-controller:${tag}" && popd
${KUSTOMIZE} build config/default > deploy/manifests/emqx-operator-controller.yaml

sed -i "s|https://github.com/emqx/emqx-operator/releases/download/.*/emqx-operator-controller.yaml|https://github.com/emqx/emqx-operator/releases/download/${tag}/emqx-operator-controller.yaml|g" docs/en_US/getting-started/getting-started.md
sed -i "s|https://github.com/emqx/emqx-operator/releases/download/.*/emqx-operator-controller.yaml|https://github.com/emqx/emqx-operator/releases/download/${tag}/emqx-operator-controller.yaml|g" docs/zh_CN/getting-started/getting-started.md
sed -i -r "s|^appVersion:.*|appVersion: ${tag}|g" deploy/charts/emqx-operator/Chart.yaml
