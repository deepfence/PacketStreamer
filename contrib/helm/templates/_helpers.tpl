{{/*
Expand the name of the chart.
*/}}
{{- define "packetstreamer.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Name of the receiver app.
*/}}
{{- define "packetstreamer-receiver.name" -}}
{{- printf "%s-receiver" (include "packetstreamer.name" .) }}
{{- end }}

{{/*
Name of the sensor app.
*/}}
{{- define "packetstreamer-sensor.name" -}}
{{- printf "%s-sensor" (include "packetstreamer.name" .) }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "packetstreamer.fullname" -}}
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
{{- define "packetstreamer.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common packetstreamer labels
*/}}
{{- define "packetstreamer.labels" -}}
helm.sh/chart: {{ include "packetstreamer.chart" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Receiver app selector labels
*/}}
{{- define "packetstreamer-receiver.selectorLabels" -}}
app.kubernetes.io/name: {{ include "packetstreamer-receiver.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Receiver app labels
*/}}
{{- define "packetstreamer-receiver.labels" -}}
helm.sh/chart: {{ include "packetstreamer.chart" . }}
{{ include "packetstreamer-receiver.selectorLabels" . }}
{{ include "packetstreamer.labels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Sensor app selector labels
*/}}
{{- define "packetstreamer-sensor.selectorLabels" -}}
app.kubernetes.io/name: {{ include "packetstreamer-sensor.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Sensor app labels
*/}}
{{- define "packetstreamer-sensor.labels" -}}
helm.sh/chart: {{ include "packetstreamer.chart" . }}
{{ include "packetstreamer-sensor.selectorLabels" . }}
{{ include "packetstreamer.labels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}
