{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "emqx-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "emqx-operator.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "emqx-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "emqx-operator.labels" -}}
helm.sh/chart: {{ include "emqx-operator.chart" . }}
{{ include "emqx-operator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "emqx-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "emqx-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "emqx-operator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "emqx-operator.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the name of the patch service account to use
*/}}
{{- define "emqx-operator.patchServiceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (cat (include "emqx-operator.fullname" .) "-" "conversion-patch" | nospace) .Values.admissionWebhooks.conversion.patch.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.admissionWebhooks.conversion.patch.serviceAccount.name }}
{{- end }}
{{- end }}


{{/*
Create the name of the certgen service account to use
*/}}
{{- define "emqx-operator.certgen.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (cat (include "emqx-operator.fullname" .) "-" "certgen" | nospace) .Values.admissionWebhooks.cert.certgen.patch.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.admissionWebhooks.cert.certgen.patch.serviceAccount.name }}
{{- end }}
{{- end }}


{{/* 
Validate values of admissionWebhooks ; - cert-manager enable or certgen enable 
*/}}
{{- define "emqx-operator.validate.admissionWebhooks" -}}
{{- if and (index .Values "admissionWebhooks" "cert" "cert-manager" "enable")  .Values.admissionWebhooks.cert.certgen.enable }}
admissionWebhooks: 
    when cert-manager and certgen are enabled at the same time, 
    use cert-mananger first
{{- end }}
{{- end }}



{{/*
Get the secretName of admissionWebhooks
*/}}
{{- define "emqx-operator.admissionWebhooks.secretName" -}}
{{- if .Values.admissionWebhooks.cert.secretName }}
{{- .Values.admissionWebhooks.cert.secretName }}
{{- else }}
{{- printf "%s-webhook-server-cert" (include "emqx-operator.fullname" .)}}
{{- end }}
{{- end }}


