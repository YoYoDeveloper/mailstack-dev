type = "controller";
bind_socket = "*:11334";
password = "mailu";
secure_ip = "{{ .Subnet }}";
{{- if .Subnet6 }}
secure_ip = "{{ .Subnet6 }}";
{{- end }}
