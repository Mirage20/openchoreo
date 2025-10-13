{{/*
Expand the name of the chart.
*/}}
{{- define "openchoreo-service.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
Uses OpenChoreo component name from context.
*/}}
{{- define "openchoreo-service.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- .Values.openchoreo.component.name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "openchoreo-service.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels applied to all resources.
These labels include OpenChoreo identity and Helm chart metadata.
*/}}
{{- define "openchoreo-service.labels" -}}
helm.sh/chart: {{ include "openchoreo-service.chart" . }}
{{ include "openchoreo-service.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
openchoreo.dev/organization: {{ .Values.openchoreo.component.organization }}
openchoreo.dev/project: {{ .Values.openchoreo.component.project }}
openchoreo.dev/component: {{ .Values.openchoreo.component.name }}
openchoreo.dev/environment: {{ .Values.openchoreo.environment.name }}
{{- end }}

{{/*
Selector labels for pods and services.
These labels are used for pod selection and should remain stable.
*/}}
{{- define "openchoreo-service.selectorLabels" -}}
app.kubernetes.io/name: {{ include "openchoreo-service.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
openchoreo.dev/component: {{ .Values.openchoreo.component.name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "openchoreo-service.serviceAccountName" -}}
{{- if .Values.pod.serviceAccountName }}
{{- .Values.pod.serviceAccountName }}
{{- else }}
{{- include "openchoreo-service.fullname" . }}
{{- end }}
{{- end }}

{{/*
Get the primary container from workload.
Returns the first container in the containers map.
*/}}
{{- define "openchoreo-service.primaryContainer" -}}
{{- $containerName := "" -}}
{{- $containerSpec := dict -}}
{{- range $name, $spec := .Values.openchoreo.workload.containers -}}
  {{- $containerName = $name -}}
  {{- $containerSpec = $spec -}}
  {{- break -}}
{{- end -}}
{{- dict "name" $containerName "spec" $containerSpec | toJson -}}
{{- end -}}

{{/*
Generate environment variables from workload container env.
*/}}
{{- define "openchoreo-service.containerEnv" -}}
{{- $primary := include "openchoreo-service.primaryContainer" . | fromJson -}}
{{- if $primary.spec.env }}
{{- range $primary.spec.env }}
- name: {{ .key }}
  value: {{ .value | quote }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Generate environment variables from workload connections.
This processes connection templates and resolves them to concrete values.
*/}}
{{- define "openchoreo-service.connectionEnv" -}}
{{- range $connName, $connSpec := .Values.openchoreo.workload.connections }}
{{- if eq $connSpec.type "api" }}
{{- if $connSpec.inject.env }}
{{- range $connSpec.inject.env }}
- name: {{ .name }}
  value: {{ .value | quote }}
  # TODO: Template resolution needs to happen in the controller before Helm rendering
  # The controller should resolve {{ .url }}, {{ .host }}, etc. from endpoint status
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Generate container ports from workload endpoints.
*/}}
{{- define "openchoreo-service.containerPorts" -}}
{{- range $name, $endpoint := .Values.openchoreo.workload.endpoints }}
- name: {{ $name | trunc 15 }}
  containerPort: {{ $endpoint.port }}
  protocol: TCP
{{- end }}
{{- end }}

{{/*
Generate Service ports from workload endpoints.
*/}}
{{- define "openchoreo-service.servicePorts" -}}
{{- range $name, $endpoint := .Values.openchoreo.workload.endpoints }}
- name: {{ $name | trunc 15 }}
  port: {{ $endpoint.port }}
  targetPort: {{ $name | trunc 15 }}
  protocol: TCP
{{- end }}
{{- end }}

{{/*
Get the namespace for deployment.
Defaults to openchoreo.namespace if set, otherwise uses Release.Namespace.
*/}}
{{- define "openchoreo-service.namespace" -}}
{{- if .Values.openchoreo.namespace }}
{{- .Values.openchoreo.namespace }}
{{- else }}
{{- .Release.Namespace }}
{{- end }}
{{- end }}

{{/*
Generate hostname for HTTPRoute based on expose level.
*/}}
{{- define "openchoreo-service.hostname" -}}
{{- $exposeLevel := . -}}
{{- if eq $exposeLevel "Project" -}}
{{ $.Values.openchoreo.component.project }}.{{ $.Values.openchoreo.component.organization }}.internal
{{- else if eq $exposeLevel "Organization" -}}
{{ $.Values.openchoreo.component.organization }}.internal
{{- else if eq $exposeLevel "Public" -}}
api.example.com
{{- end -}}
{{- end -}}

{{/*
Check if any API has a specific expose level.
*/}}
{{- define "openchoreo-service.hasExposeLevel" -}}
{{- $level := .level -}}
{{- $found := false -}}
{{- range $apiName, $apiSpec := .root.Values.openchoreo.apis -}}
{{- if $apiSpec.rest.exposeLevels -}}
{{- if has $level $apiSpec.rest.exposeLevels -}}
{{- $found = true -}}
{{- end -}}
{{- end -}}
{{- end -}}
{{- $found -}}
{{- end -}}

{{/*
Get image pull secrets list.
*/}}
{{- define "openchoreo-service.imagePullSecrets" -}}
{{- if .Values.imagePullSecrets.enabled }}
{{- range .Values.imagePullSecrets.secrets }}
- name: {{ .name }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Merge additional environment variables with workload env and connection env.
*/}}
{{- define "openchoreo-service.allEnv" -}}
{{- include "openchoreo-service.containerEnv" . }}
{{- include "openchoreo-service.connectionEnv" . }}
{{- if .Values.env }}
{{- toYaml .Values.env }}
{{- end }}
{{- end }}
