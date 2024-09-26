{{- define "gen-selfsigned" -}}

{{- $cert := genSelfSignedCert .certificateCN nil (list (printf "%s.%s" .certificateCN "svc")) (int .certificateValidityDays) }}

{{- $_ := set . "certificate" $cert.Cert }}
{{- $_ := set . "privateKey" $cert.Key }}

{{- end }}
