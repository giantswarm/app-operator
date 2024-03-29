{{/* vim: set filetype=mustache: */}}
{{/*
Create a name stem for resource names

When pods for deployments are created they have an additional 16 character
suffix appended, e.g. "-957c9d6ff-pkzgw". Given that Kubernetes allows 63
characters for resource names, the stem is truncated to 47 characters to leave
room for such suffix.
*/}}
{{- define "resource.default.name" -}}
{{- .Release.Name | replace "." "-" | trunc 47 | trimSuffix "-" -}}
{{- end -}}

{{- define "resource.chart.name" -}}
{{- include "resource.default.name" . -}}-chart
{{- end -}}

{{- define "resource.psp.name" -}}
{{- include "resource.default.name" . -}}-psp
{{- end -}}

{{- define "resource.pullSecret.name" -}}
{{- include "resource.default.name" . -}}-pull-secret
{{- end -}}

{{- define "resource.default.namespace" -}}
{{-  .Release.Namespace | quote }}
{{- end -}}

{{/*
The deployment of app-operator for management cluster reconciles a special app
CR version of 0.0.0.
*/}}
{{- define "resource.app.unique" -}}
{{- if eq $.Chart.Name $.Release.Name }}true{{ else }}false{{ end }}
{{- end -}}
{{- define "resource.app.version" -}}
{{- if eq $.Chart.Name $.Release.Name }}0.0.0{{ else }}{{ .Chart.AppVersion }}{{ end }}
{{- end -}}

{{- define "resource.vpa.enabled" -}}
{{- if and (.Capabilities.APIVersions.Has "autoscaling.k8s.io/v1") (.Values.verticalPodAutoscaler.enabled) }}true{{ else }}false{{ end }}
{{- end -}}

{{/*
The unique deployment in the management cluster requires more resources than
the per workload cluster instances.
*/}}
{{- define "resource.deployment.resources" -}}
{{- if eq (include "resource.app.unique" .) "true" -}}
requests:
{{ toYaml .Values.deployment.management.requests | indent 2 -}}
{{ if eq (include "resource.vpa.enabled" .) "false" }}
limits:
{{ toYaml .Values.deployment.management.limits | indent 2 -}}
{{- end -}}
{{- else }}
requests:
{{ toYaml .Values.deployment.workload.requests | indent 2 -}}
{{ if eq (include "resource.vpa.enabled" .) "false" }}
limits:
{{ toYaml .Values.deployment.workload.limits | indent 2 -}}
{{- end -}}
{{- end -}}
{{- end -}}
