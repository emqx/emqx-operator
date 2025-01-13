#!/usr/bin/env bash
set -euo pipefail

# ensure dir
project_dir="$(dirname "$(readlink -f "$0")")"/..
cd -P -- "$project_dir"

# ensure kustomize
test -s "$project_dir/bin/kustomize" || make kustomize -C "$project_dir"

dir=$(mktemp -d)
# generate CRDs
pushd "$dir"
"$project_dir/bin/kustomize" build "$project_dir/config/crd" > crds.yaml
yq -s '"crd." + .metadata.name + ".yaml"' crds.yaml
popd

while IFS= read -r -d '' file
do
	if yq '.metadata.annotations' "$file" | yq 'keys' | grep -q 'cert-manager.io/inject-ca-from' > /dev/null 2>&1 ; then
		yq -i '.metadata.annotations."cert-manager.io/inject-ca-from" = "{{ .Release.Namespace }}/{{ include \"emqx-operator.fullname\" . }}-serving-cert"' "$file"
		yq -i '.spec.conversion.webhook.clientConfig.service.name= "{{ include \"emqx-operator.fullname\" . }}-webhook-service"' "$file"
		yq -i '.spec.conversion.webhook.clientConfig.service.namespace = "{{ .Release.Namespace }}"' "$file"
  fi

    sed -i '1i {{- if not .Values.skipCRDs }}\n' "$file"
	echo -e '\n{{- end }}' >> "$file"
done <   <(find "$dir" -depth -type f -name "*.yaml" ! -name "crds.yaml" -print0)

find "$dir" -depth -type f -name "*.yaml" ! -name "crds.yaml" -exec mv {} "$project_dir/deploy/charts/emqx-operator/templates" \;
