#!/usr/bin/env bash
set -euo pipefail

tag=$1

APIS_DIR='apis/apps'
REFERENCE_OUTPUT='docs/en_US/reference'
REFERENCE_CONFIG='crd-reference-config.yaml'


# ensure dir
cd -P -- "$(dirname -- "${BASH_SOURCE[0]}")/.."


make kustomize
KUSTOMIZE=${PWD}/bin/kustomize
pushd config/manager && ${KUSTOMIZE} edit set image controller="emqx/emqx-operator-controller:${tag}" && popd
${KUSTOMIZE} build config/default > deploy/manifests/emqx-operator-controller.yaml

# Default case for GUN sed, use "sed -i"
SED_REPLACE="sed -i "
case $(sed --help 2>&1) in
    *GNU*) SED_REPLACE="sed -i ";;
    *BusyBox*) SED_REPLACE="sed -i ";;
    *) SED_REPLACE="sed -i '' ";;
esac

${SED_REPLACE} "s|https://github.com/emqx/emqx-operator/releases/download/.*/emqx-operator-controller.yaml|https://github.com/emqx/emqx-operator/releases/download/${tag}/emqx-operator-controller.yaml|g" docs/en_US/getting-started/getting-started.md
${SED_REPLACE} "s|https://github.com/emqx/emqx-operator/releases/download/.*/emqx-operator-controller.yaml|https://github.com/emqx/emqx-operator/releases/download/${tag}/emqx-operator-controller.yaml|g" docs/zh_CN/getting-started/getting-started.md
${SED_REPLACE} -r "s|^appVersion:.*|appVersion: ${tag}|g" deploy/charts/emqx-operator/Chart.yaml

# update reference for apis

function updateCrdReference {
    for API_DIR in $(find apis/apps -type d -d 1); do
        crd-ref-docs --source-path=${PWD}/${API_DIR} --config=${REFERENCE_CONFIG} --output-path=${PWD}/${REFERENCE_OUTPUT} --renderer=markdown && mv ${PWD}/${REFERENCE_OUTPUT}/out.md ${PWD}/${REFERENCE_OUTPUT}/$(basename ${API_DIR})-reference.md
    done
}

if type crd-ref-docs >/dev/null 2>&1
then
    updateCrdReference
else
    echo "crd-ref-docs not exist"
    echo "please refer to the documentation https://github.com/elastic/crd-ref-docs for installation"
    exit 1
fi
