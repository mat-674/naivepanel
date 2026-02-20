package naiveproxy

import (
	"bytes"
	"fmt"
	"naivepanel/internal/models"
	"strings"
	"text/template"
)

const caddyfileTemplate = `{
	order forward_proxy before reverse_proxy
{{- if .TLSEmail }}
	acme_ca https://acme-v02.api.letsencrypt.org/directory
{{- end }}
}

{{- if .Domain }}
:{{ .Port }}, {{ .Domain }} {
{{- else }}
:{{ .Port }} {
{{- end }}
{{- if .TLSEmail }}
	tls {{ .TLSEmail }}
{{- else }}
	tls internal
{{- end }}

	route {
		forward_proxy {
{{- range .Users }}
			basic_auth {{ .Username }} {{ .Password }}
{{- end }}
			hide_ip
			hide_via
			probe_resistance
		}

{{- if .DecoySite }}
		reverse_proxy {{ .DecoySite }} {
			header_up Host {upstream_hostport}
		}
{{- else }}
		respond "Hello, World!" 200
{{- end }}
	}
}
`

// CaddyfileData contains all data needed to generate a Caddyfile
type CaddyfileData struct {
	Domain    string
	Port      int
	TLSEmail  string
	DecoySite string
	Users     []models.ProxyUser
}

// GenerateCaddyfile generates a Caddyfile from settings and users
func GenerateCaddyfile(settings *models.Settings, users []models.ProxyUser) (string, error) {
	data := CaddyfileData{
		Domain:    settings.Domain,
		Port:      settings.Port,
		TLSEmail:  settings.TLSEmail,
		DecoySite: settings.DecoySite,
		Users:     users,
	}

	if data.Port == 0 {
		data.Port = 443
	}

	tmpl, err := template.New("caddyfile").Parse(caddyfileTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse caddyfile template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute caddyfile template: %w", err)
	}

	// Clean up extra blank lines
	result := buf.String()
	for strings.Contains(result, "\n\n\n") {
		result = strings.ReplaceAll(result, "\n\n\n", "\n\n")
	}

	return result, nil
}
