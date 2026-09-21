{{- define "avatars.fullname" -}}
{{- .Release.Name -}}
{{- end -}}

{{- define "avatars.databaseDSN" -}}
{{- printf "postgres://%s:%s@%s:%d/%s?sslmode=disable" .Values.database.user .Values.database.password .Values.external.host (.Values.external.postgresPort | int) .Values.database.name -}}
{{- end -}}

{{- define "avatars.rabbitmqURL" -}}
{{- printf "amqp://%s:%s@%s:%d/" .Values.rabbitmq.user .Values.rabbitmq.password .Values.external.host (.Values.external.rabbitmqPort | int) -}}
{{- end -}}

{{- define "avatars.s3Endpoint" -}}
{{- printf "http://%s:%d" .Values.external.host (.Values.external.minioPort | int) -}}
{{- end -}}
