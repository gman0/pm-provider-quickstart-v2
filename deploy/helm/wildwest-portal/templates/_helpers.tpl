{{- define "wildwest-portal.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "wildwest-portal.fullname" -}}
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

{{- define "wildwest-portal.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "wildwest-portal.labels" -}}
helm.sh/chart: {{ include "wildwest-portal.chart" . }}
{{ include "wildwest-portal.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "wildwest-portal.selectorLabels" -}}
app.kubernetes.io/name: {{ include "wildwest-portal.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "wildwest-portal.image" -}}
{{- $tag := .Values.image.tag | default .Chart.AppVersion -}}
{{- if hasPrefix "@" $tag -}}
{{- printf "%s%s" .Values.image.repository $tag -}}
{{- else -}}
{{- printf "%s:%s" .Values.image.repository $tag -}}
{{- end -}}
{{- end }}

{{- define "wildwest-portal.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "wildwest-portal.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}
