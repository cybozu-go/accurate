{{/*
Expand the name of the chart.
*/}}
{{- define "accurate.name" -}}
{{- default .Chart.Name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "accurate.fullname" -}}
{{- $name := default .Chart.Name }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "accurate.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "accurate.labels" -}}
helm.sh/chart: {{ include "accurate.chart" . }}
app.kubernetes.io/name: {{ include "accurate.name" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Check that the user has not unset both .installCRDs and .crds.enabled or
set .installCRDs and disabled .crds.keep.
.installCRDs is deprecated and users should use .crds.enabled and .crds.keep instead.
*/}}
{{- define "accurate.crd-check" -}}
  {{- if and ( not .Values.installCRDs) (not .Values.crds.enabled) }}
    {{- fail "ERROR: the deprecated .installCRDs option cannot be disabled at the same time as its replacement .crds.enabled" }}
  {{- end }}
  {{- if and (.Values.installCRDs) (not .Values.crds.keep) }}
    {{- fail "ERROR: .crds.keep is not compatible with .installCRDs, please use .crds.enabled and .crds.keep instead" }}
  {{- end }}
{{- end -}}
