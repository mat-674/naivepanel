package naiveproxy

import (
	"bytes"
	"fmt"
	"naivepanel/internal/models"
	"net"
	"net/mail"
	"net/url"
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

	# The panel itself is bound to loopback. This is the only public route to
	# the subscription API, and it keeps the original request URI intact.
	handle /{{ .SubscriptionPath }}/* {
		reverse_proxy {{ .PanelUpstream }}
	}

	# All remaining requests are handled by NaiveProxy or the configured decoy.
	handle {
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
}
`

// CaddyfileData contains all data needed to generate a Caddyfile
type CaddyfileData struct {
	Domain    string
	Port      int
	TLSEmail  string
	DecoySite string
	Users     []models.ProxyUser
	// SubscriptionPath is a sanitized path prefix without leading slashes.
	SubscriptionPath string
	// PanelUpstream is the loopback address of the administrative panel.
	PanelUpstream string
}

// GenerateCaddyfile generates a Caddyfile from settings and users
func GenerateCaddyfile(settings *models.Settings, users []models.ProxyUser, panelUpstream string) (string, error) {
	if settings == nil {
		return "", fmt.Errorf("settings are required")
	}

	subscriptionPath, err := NormalizeSubscriptionPath(settings.SubPath)
	if err != nil {
		return "", err
	}
	domain, err := NormalizeDomain(settings.Domain)
	if err != nil {
		return "", err
	}
	tlsEmail, err := NormalizeTLSEmail(settings.TLSEmail)
	if err != nil {
		return "", err
	}
	decoySite, err := NormalizeDecoySite(settings.DecoySite)
	if err != nil {
		return "", err
	}
	if _, _, err := net.SplitHostPort(panelUpstream); err != nil {
		return "", fmt.Errorf("invalid panel upstream %q: %w", panelUpstream, err)
	}

	// Backstop: credentials are written verbatim into basic_auth tokens, so a
	// malformed username/password (space, newline, quote) would corrupt the
	// Caddyfile and break every user's proxy. Reject before emitting.
	for _, u := range users {
		if err := ValidateProxyUsername(u.Username); err != nil {
			return "", fmt.Errorf("user %q: %w", u.Username, err)
		}
		if err := ValidateProxyPassword(u.Password); err != nil {
			return "", fmt.Errorf("user %q: %w", u.Username, err)
		}
	}

	data := CaddyfileData{
		Domain:           domain,
		Port:             settings.Port,
		TLSEmail:         tlsEmail,
		DecoySite:        decoySite,
		Users:            users,
		SubscriptionPath: subscriptionPath,
		PanelUpstream:    panelUpstream,
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

// NormalizeSubscriptionPath returns a safe, relative path prefix for the
// public subscription endpoint. A single prefix may contain multiple path
// segments, but each segment is restricted to URL-safe identifier characters.
func NormalizeSubscriptionPath(path string) (string, error) {
	path = strings.Trim(strings.TrimSpace(path), "/")
	if path == "" {
		return "sub", nil
	}
	if len(path) > 128 {
		return "", fmt.Errorf("subscription path must be at most 128 characters")
	}

	for _, segment := range strings.Split(path, "/") {
		if segment == "" || !isSafePathSegment(segment) {
			return "", fmt.Errorf("subscription path contains an invalid segment")
		}
	}

	return path, nil
}

func isSafePathSegment(segment string) bool {
	for _, r := range segment {
		if (r < 'a' || r > 'z') &&
			(r < 'A' || r > 'Z') &&
			(r < '0' || r > '9') &&
			r != '-' && r != '_' {
			return false
		}
	}
	return true
}

// NormalizeDomain accepts a DNS name (or an IP address for local testing) as
// a Caddy site address. Schemes, paths and Caddyfile control characters are
// intentionally rejected because the port is configured separately.
func NormalizeDomain(domain string) (string, error) {
	domain = strings.TrimSuffix(strings.TrimSpace(domain), ".")
	if domain == "" {
		return "", nil
	}
	if net.ParseIP(domain) != nil {
		return domain, nil
	}
	if len(domain) > 253 {
		return "", fmt.Errorf("domain is too long")
	}

	for _, label := range strings.Split(domain, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return "", fmt.Errorf("domain is invalid")
		}
		for _, r := range label {
			if (r < 'a' || r > 'z') &&
				(r < 'A' || r > 'Z') &&
				(r < '0' || r > '9') &&
				r != '-' {
				return "", fmt.Errorf("domain is invalid")
			}
		}
	}

	return strings.ToLower(domain), nil
}

// NormalizeTLSEmail validates the email token passed to Caddy's tls directive.
func NormalizeTLSEmail(email string) (string, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return "", nil
	}

	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email {
		return "", fmt.Errorf("TLS email is invalid")
	}

	return email, nil
}

// NormalizeDecoySite accepts a single HTTPS upstream without a path, query or
// credentials. That matches the Caddy reverse_proxy upstream syntax and keeps
// untrusted settings from being interpreted as Caddyfile directives.
func NormalizeDecoySite(site string) (string, error) {
	site = strings.TrimSpace(site)
	if site == "" {
		return "", nil
	}

	parsed, err := url.ParseRequestURI(site)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil ||
		(parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("decoy site must be an HTTPS origin without a path")
	}

	return "https://" + parsed.Host, nil
}

// Proxy credential length bounds.
const (
	maxProxyUsernameLen = 64
	maxProxyPasswordLen = 128
)

// ValidateProxyUsername rejects usernames that would be unsafe in the
// generated Caddyfile or the naive+https:// connection URI. Credentials are
// written verbatim into a space-separated `basic_auth` token and into the
// userinfo component of a URI, so they are restricted to the RFC 3986
// unreserved set, which is safe in both contexts without escaping.
func ValidateProxyUsername(username string) error {
	if username == "" {
		return fmt.Errorf("proxy username is required")
	}
	if len(username) > maxProxyUsernameLen {
		return fmt.Errorf("proxy username must be at most %d characters", maxProxyUsernameLen)
	}
	if !isUnreservedString(username) {
		return fmt.Errorf("proxy username may contain only letters, digits, and the characters - . _ ~")
	}
	return nil
}

// ValidateProxyPassword applies the same character policy to passwords.
func ValidateProxyPassword(password string) error {
	if password == "" {
		return fmt.Errorf("proxy password is required")
	}
	if len(password) > maxProxyPasswordLen {
		return fmt.Errorf("proxy password must be at most %d characters", maxProxyPasswordLen)
	}
	if !isUnreservedString(password) {
		return fmt.Errorf("proxy password may contain only letters, digits, and the characters - . _ ~")
	}
	return nil
}

// isUnreservedString reports whether s consists solely of RFC 3986 unreserved
// characters: ALPHA / DIGIT / "-" / "." / "_" / "~".
func isUnreservedString(s string) bool {
	for _, r := range s {
		if (r < 'a' || r > 'z') &&
			(r < 'A' || r > 'Z') &&
			(r < '0' || r > '9') &&
			r != '-' && r != '.' && r != '_' && r != '~' {
			return false
		}
	}
	return true
}
