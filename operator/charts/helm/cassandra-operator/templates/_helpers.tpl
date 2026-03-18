{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "helm-chart.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "helm-chart.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "helm-chart.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "helm-chart.labels" -}}
helm.sh/chart: {{ include "helm-chart.chart" . }}
{{ include "helm-chart.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Selector labels
*/}}
{{- define "helm-chart.selectorLabels" -}}
app.kubernetes.io/name: {{ include "helm-chart.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/*
Create the name of the serviceList account to use
*/}}
{{- define "helm-chart.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
    {{ default (include "helm-chart.fullname" .) .Values.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.serviceAccount.name }}
{{- end -}}
{{- end -}}

{{/*
[NoSQL Operator Core] Vault secret template
Arguments:
Dictionary with:
1. "vlt" - .vaultRegistration section
2. "secret" section includes next elements:
    .secretName (required)
    .password (required)
    .username (optional)
    .vaultPasswordPath (optional)
3. "isInternal" is a required boolean parameter
Usage example:
{{template "nosql.core.secret.vault" (dict "vltEnabled" .Values.vaultRegistration "vltPath" "kv_path" "secret" .Values.cassandra )}}
*/}}
{{- define "nosql.core.secret.vault" -}}
{{ $_ := set . "userEnv" "" }}
{{ $_ := set . "userPass" "" }}
{{include "nosql.core.secret.vault.fromEnv" $_ }}
{{- end -}}

{{/*
[NoSQL Operator Core] Vault secret template
Arguments:
Dictionary with:
1. "vlt" - .vaultRegistration section
2. "secret" section includes next elements:
    .secretName (required)
    .password (required)
    .username (optional)
    .vaultPasswordPath (optional)
3. "isInternal" is a required boolean parameter
Usage example:
{{template "nosql.core.secret.vault.fromEnv" (dict "vltEnabled" .Values.vaultRegistration "vltPath" "kv_path" "secret" .Values.cassandra "userEnv" .Values.INFRA_CASSANDRA_USERNAME "passEnv" .Values.INFRA_CASSANDRA_PASSWORD )}}
*/}}
{{- define "nosql.core.secret.vault.fromEnv" -}}
apiVersion: v1
kind: Secret
metadata:
  name: {{ .secret.secretName }}
  labels:
    {{ include "cassandra.defaultLabels" .values | nindent 4 }}
stringData:
  {{- if .vltEnabled }}
    {{- if .secret.vaultPasswordPath }}
  password: {{ .secret.vaultPasswordPath | quote }}
    {{- else }}
        {{- if (.isInternal) }}
  password: 'vault:{{ .vltPath }}'
        {{- else }}
  password: {{ include "fromEnv" (dict "envName" .passEnv "default" .secret.password) | quote }}
        {{- end }}
    {{- end }}
  {{- else }}
  password: {{ include "fromEnv" (dict "envName" .passEnv "default" .secret.password) | quote }}
  {{- end }}
  {{- if .secret.username }}
  username: {{ include "fromEnv" (dict "envName" .userEnv "default" .secret.username) | quote }} 
  {{- end }}
type: Opaque
{{- end -}}

{{/*
[NoSQL Operator Core] Internal secret template
{{template "nosql.core.secret.internal" (dict "vlt" .Values.vaultRegistration "secret" .Values.redis)}}
*/}}
{{- define "nosql.core.secret.internal" -}}
{{include "nosql.core.secret.vault" (set . "isInternal" true)}}
{{- end -}}

{{/*
[NoSQL Operator Core] External secret template
{{template "nosql.core.secret.external" (dict "vlt" .Values.vaultRegistration "secret" .Values.redis)}}
*/}}
{{- define "nosql.core.secret.external" -}}
{{ include "nosql.core.secret.vault" (set . "isInternal" false) }}
{{- end -}}

{{/*
[NoSQL Operator Core] PodDisruptionBudget
Dictionary with:
1. "name" - pdb name
2. "labels" - label selectors map
3. "minAvailable" - desired pods count
{{template "nosql.core.pdb" (dict "name" "cassandra" "labels" $labels "minAvailable" $minAvailable)}}
*/}}
{{- define "nosql.core.pdb" -}}
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: {{ .name | quote }}
  labels:
    {{ include "cassandra.defaultLabels" . | nindent 4 }}
spec:
  minAvailable: {{ .minAvailable }}
  selector:
    matchLabels:
      {{- range $k, $v := .labels }}
      {{ $k | quote }}: {{ $v | quote }}
      {{- end }}
{{- end -}}

{{/*
[NoSQL Operator Core] Create the name for service registration in Consul
Dictionary with:
1. "name" - the service name
2. "default" - default name that will be converted to %default%-%namespace% name if the "name" field is empty
3. "namespace" - namespace for default name
{{template "nosql.core.consul.serviceName" (dict "name" "cassandra-custom" "default" "cassandra")}}
*/}}
{{- define "nosql.core.consul.serviceName" -}}
  {{ (.name) | default (printf "%s-%s" .namespace .default) }}
{{- end -}}

{{/*
[Cassandra Operator Core] Parses storage section
Dictionary with:
1. "dc" - the datacenter
2. "storage" - storage section
{{template "nosql.cassandra.storage" (dict "dc" $dc "storage" $storage "defaultSC" .Values.STORAGE_RWO_CLASS }}
*/}}
{{- define "nosql.cassandra.storage" -}}
{{- if eq "string" (printf "%T" .storage.size) }}
- size:
    - {{ .storage.size }}
{{- else }}
- size:
{{- range $key, $value := .storage.size }}
    - {{ $value | quote }}
 {{- end }}
{{- end }}
  waitPvcBound: {{if or (not .storage.waitPvcBound) (eq (.storage.waitPvcBound | toString) "<nil>") }}false{{ else }}true{{ end }}
{{- if .storage.mountSettings }}
  mountSettings:
    name: {{ .storage.mountSettings.name | quote }}
    mountPath: {{ .storage.mountSettings.mountPath | quote }}
    {{- if .storage.mountSettings.subPath }}
    subPath: {{ .storage.mountSettings.subPath | quote }}
    {{- end -}}
{{- else }}
  mountSettings:
    name: "data"
    mountPath: "/var/lib/cassandra/data"
{{- end }}
{{- if .storage.volumes }}
  volumes:
  {{- range $key, $value := .storage.volumes }}
    - {{ $value | quote }}
{{- end }}
{{- end }}
{{- if .storage.nodeLabels }}
  nodeLabels:
  {{- range $key, $value := .storage.nodeLabels }}
  {{- $arraystart := "-" -}}
  {{- range $k, $v := $value }}
    {{ $arraystart }} {{ $k | quote }}: {{ $v | quote }}
  {{- $arraystart = " " }}
  {{- end }}
  {{- end }}
{{- end }}
{{- if .storage.storageClasses }}
  storageClasses:
  {{- range $key, $value := .storage.storageClasses }}
    - {{ $value | quote }}
  {{- end }}
{{- else if .dotEnvSc }}
  storageClasses:
    - {{ .dotEnvSc | quote }}
{{- end }}
{{- if .storage.matchLabelSelectors }}
  matchLabelSelectors:
  {{- range $key, $value := .storage.matchLabelSelectors }}
  {{- $arraystart := "-" -}}
  {{- range $k, $v := $value }}
    {{ $arraystart }} {{ $k | quote }}: {{ $v | quote }}
  {{- $arraystart = " " }}
  {{- end }}
{{- end }}
{{- end }}
{{- end -}}


{{- define "deployment.apiVersion" -}}
  {{- if semverCompare "<1.9-0" .Capabilities.KubeVersion.GitVersion -}}
    {{- print "apps/v1beta2" -}}
  {{- else -}}
    {{- print "apps/v1" -}}
  {{- end -}}
{{- end -}}

{{/*
[Cassandra Operator Core] Docker image
Dictionary with:
1. "deployName" - deploy-param from description.yaml
2. "SERVICE_NAME" - name of service with git group and git repo
3. "vals" - .Values
4.  "default" - default docker image
{{template "find_image" (dict "deployName" "cassandraOperator" "SERVICE_NAME" "cassandra-operator" "vals" .Values "default" .Values.operator.dockerImage) }}
*/}}

{{- define "find_image" -}}
  {{- $image := .default -}}

  {{- if .vals.deployDescriptor -}}
    {{- if index .vals.deployDescriptor .deployName -}}
      {{- $image = (index .vals.deployDescriptor .deployName "image") -}}
    {{- else if index .vals.deployDescriptor .SERVICE_NAME -}}
      {{- $image = (index .vals.deployDescriptor .SERVICE_NAME "image") -}}
    {{- end -}}
  {{- end -}}

  {{ printf "%s" $image }}
{{- end -}}


{{/*
[Cassandra Operator Core] returns value from ENV if it exists there, otherwise from default
Dictionary with:
1. "envName" - name of env var to get value from
2.  "default" - default value from values.yaml
{{template "fromEnv" (dict "envName" ".Values.VAULT_ADDR" "default" .Values.vaultRegistration.token) }}
*/}}
{{- define "fromEnv" -}}
  {{- $envValue := .envName -}}
{{- if and (ne ($envValue | toString) "<nil>") (ne ($envValue | toString) "") -}}
    {{- .envName -}}
  {{- else -}}
    {{- .default -}}
  {{- end -}}
{{- end -}}


{{/*
Dictionary with:
Uses value from values.yaml if defined, otherwise value from environment variable if defined, else - default
1. "dotVar" - parameter defined with dots like dbaas.install
2. "enVar" - parameter defined as environment variable like DBAAS_ENABLED
3.  "default" - default value
{{template "fromValuesThenEnvElseDefault" (dict "dotVar" .Values.dbaas.install "envVar" .Values.DBAAS_ENABLED "default" true ) }}
*/}}
{{- define "fromValuesThenEnvElseDefault" -}}
  {{- if and (ne (.dotVar | toString) "<nil>") (ne (.dotVar | toString) "") -}}
    {{- .dotVar -}}
  {{- else if and (ne (.envVar | toString) "<nil>") (ne (.envVar | toString) "") -}}
    {{- .envVar -}}
  {{- else -}}
    {{- .default -}}
  {{- end -}}
{{- end -}}

{{/*
[Cassandra Operator Core] from env of from values
Dictionary with:
1. "envName" - name of env var to get value from
2.  "default" - default value from values.yaml
{{template "ifEnvThenDefault" (dict "envName" .Values.VAULT_ADDR "then" (printf %s_%s .Values.VAULT_ADDR "const" ) "default" .Values.vaultRegistration.token) }}
*/}}
{{- define "ifEnvThenDefault" -}}
  {{- $value := .default -}}
  {{- if .envName -}}
    {{- $value = .then -}}
  {{- else -}}
    {{- $value = .default -}}
  {{- end -}}
  {{- if $value -}}
  {{ printf "%s" $value }}
  {{- end -}}
{{- end -}}

{{/*
DNS names used to generate SSL certificate with "Subject Alternative Name" field
*/}}
{{- define "dbaasAdapter.certDnsNames" -}}
  {{- $dnsNames := list "localhost" "dbaas-cassandra-adapter" (printf "%s.%s" "dbaas-cassandra-adapter" .Release.Namespace) (printf "%s.%s.svc" "dbaas-cassandra-adapter" .Release.Namespace) -}}
  {{- $dnsNames = concat $dnsNames .Values.tls.generateCerts.subjectAlternativeName.additionalDnsNames -}}
  {{- $dnsNames | toYaml -}}
{{- end -}}
{{/*
IP addresses used to generate SSL certificate with "Subject Alternative Name" field
*/}}
{{- define "backupDaemon.certDnsNames" -}}
  {{- $dnsNames := list "localhost" "cassandra-backup-daemon" (printf "%s.%s" "cassandra-backup-daemon" .Release.Namespace) (printf "%s.%s.svc" "cassandra-backup-daemon" .Release.Namespace) -}}
  {{- $dnsNames = concat $dnsNames .Values.tls.generateCerts.subjectAlternativeName.additionalDnsNames -}}
  {{- $dnsNames | toYaml -}}
{{- end -}}

{{- define "cassandra.certDnsNames" -}}
  {{- $dnsNames := list "localhost" "cassandra" (printf "%s.%s" "cassandra" .Release.Namespace) (printf "%s.%s.svc" "cassandra" .Release.Namespace) -}}
  {{- $dnsNames = concat $dnsNames .Values.tls.generateCerts.subjectAlternativeName.additionalDnsNames -}}
  {{- $dnsNames | toYaml -}}
{{- end -}}
{{/*
IP addresses used to generate SSL certificate with "Subject Alternative Name" field
*/}}
{{- define "common.certIpAddresses" -}}
  {{- $ipAddresses := list "127.0.0.1" -}}
  {{- $ipAddresses = concat $ipAddresses .Values.tls.generateCerts.subjectAlternativeName.additionalIpAddresses -}}
  {{- $ipAddresses | toYaml -}}
{{- end -}}


{{/*
TLS Static Metric secret template
Arguments:
Dictionary with:
* "namespace" is a namespace of application
* "application" is name of application
* "service" is a name of service
* "enabledSsl" is ssl enabled for service
* "secret" is a name of tls secret for service
* "certProvider" is a type of tls certificates provider
* "certificate" is a name of CertManger's Certificate resource for service
Usage example:
{{template "global.tlsStaticMetric" (dict "namespace" .Release.Namespace "application" .Chart.Name "service" .global.name "enabledSsl" (include "global.sslEnabled" .) "secret" (include "global.sslSecretName" .) "certProvider" (include "services.certProvider" .) "certificate" (printf "%s-tls-certificate" (include "global.name")) }}
*/}}
{{- define "global.tlsStaticMetric" -}}
- expr: {{ ternary "1" "0" .enabledSsl }}
  labels:
    namespace: "{{ .namespace }}"
    application: "{{ .application }}"
    service: "{{ .service }}"
    {{ if .enabledSsl }}
    secret: "{{ .secret }}"
    {{ if eq .certProvider "cert-manager" }}
    certificate: "{{ .certificate }}"
    {{ end }}
    {{ end }}
  record: service:tls_status:info
{{- end -}}


{{- define "getCassandraResourcesForProfile" -}}
  {{- $flavor := .dotVar }}
{{- if and (ne (.envVar | toString) "<nil>") (ne (.envVar | toString) "") -}}
  {{- $flavor = .envVar -}}
{{- end -}}
  {{- if eq $flavor "small" }}
    resources:
      requests:
        cpu: 250m
        memory: 1Gi
      limits:
        cpu: 500m
        memory: 2Gi
  {{- else if eq $flavor "medium" }}
    resources:
      requests:
        cpu: 1
        memory: 2Gi
      limits:
        cpu: 2
        memory: 4Gi
  {{- else if eq $flavor "large" }}
    resources:
      requests:
        cpu: 2
        memory: 4Gi
      limits:
        cpu: 8
        memory: 16Gi
  {{- else if $flavor -}}
  {{- fail "value for .Values.global.profile is not one of  `small`, `medium`, `large`" }}
  {{- else }}
    resources:
      requests:
        cpu: {{ .values.cassandra.resources.requests.cpu | quote }}
        memory: {{ .values.cassandra.resources.requests.memory }}
      limits:
        cpu: {{ .values.cassandra.resources.limits.cpu | quote }}
        memory: {{ .values.cassandra.resources.limits.memory }}
  {{- end -}}
{{- end -}}

{{- define "getBackupResourcesForProfile" -}}
  {{- $flavor := .dotVar }}
{{- if and (ne (.envVar | toString) "<nil>") (ne (.envVar | toString) "") -}}
  {{- $flavor = .envVar -}}
{{- end -}}
  {{- if eq $flavor "small" }}
    resources:
      requests:
        cpu: 150m
        memory: 256Mi
      limits:
        cpu: 500m
        memory: 512Mi
  {{- else if eq $flavor "medium" }}
    resources:
      requests:
        cpu: 150m
        memory: 256Mi
      limits:
        cpu: 1
        memory: 1Gi
  {{- else if eq $flavor "large" }}
    resources:
      requests:
        cpu: 150m
        memory: 256Mi
      limits:
        cpu: 2
        memory: 2Gi
  {{- else if $flavor -}}
  {{- fail "value for .Values.global.profile is not one of  `small`, `medium`, `large`" }}
  {{- else }}
    resources:
      requests:
        cpu: {{ .values.backupDaemon.resources.requests.cpu | quote }}
        memory: {{ .values.backupDaemon.resources.requests.memory }}
      limits:
        cpu: {{ .values.backupDaemon.resources.limits.cpu | quote }}
        memory: {{ .values.backupDaemon.resources.limits.memory }}
  {{- end -}}
{{- end -}}

{{/*
Common Cassandra resources labels
*/}}
{{- define "cassandra.defaultLabels" -}}
{{- if .Values.ARTIFACT_DESCRIPTOR_VERSION }}
app.kubernetes.io/version: {{ default "" .Values.ARTIFACT_DESCRIPTOR_VERSION | trunc 63 | trimAll "-_." }}
{{- end }}
app.kubernetes.io/part-of: {{ default "cassandra" .Values.PART_OF }}
app.kubernetes.io/managed-by: {{ default "operator" .Values.MANAGED_BY }}
{{- end -}}

{{- define "cassandra.monitoredImages" -}}
  {{- if .Values.deployDescriptor -}}
    {{- printf "deployment cassandra-operator cassandra-operator %s, " (include "find_image" (dict "deployName" "dockerCassandraOperator" "SERVICE_NAME" "dockerCassandraOperator" "vals" .Values "default" "not_found")) -}}
    {{- if .Values.cassandra.install -}}
      {{- printf "statefulset cassandra0 cassandra0 %s, " (include "find_image" (dict "deployName" "cassandra" "SERVICE_NAME" "cassandra" "vals" .Values "default" "not_found")) -}}
      {{- if .Values.reaper.install -}}
        {{- printf "statefulset cassandra0 cassandra-reaper %s, " (include "find_image" (dict "deployName" "cassandra-reaper" "SERVICE_NAME" "cassandra-reaper" "vals" .Values "default" "not_found")) -}}
      {{- end -}}
    {{- end -}}
  {{- end -}}
{{- end -}}
