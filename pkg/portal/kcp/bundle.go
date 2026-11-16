package kcp

import (
	"bytes"
	"strings"
	"text/template"
)

const ProviderName = "wildwest"

const bundleTpl = `---
apiVersion: v1
kind: Secret
metadata:
  name: {{ .IdentityName }}-token
  namespace: default
type: Opaque
stringData:
  accessToken: {{ .AccessToken }}
---
apiVersion: providers.platform-mesh.io/v1alpha1
kind: Provider
metadata:
  name: {{ .ProviderName }}
{{- if .CABundle }}
spec:
  remote:
    caBundle: |
{{ indent 6 .CABundle }}
{{- else }}
spec:
  remote: {}
{{- end }}
---
apiVersion: providers.platform-mesh.io/v1alpha1
kind: ProviderAccessRequest
metadata:
  name: {{ .ProviderName }}
spec:
  providerRef:
    name: {{ .ProviderName }}
  accessTokenSecretRef:
    name: {{ .IdentityName }}-token
    namespace: default
    key: accessToken
`

type BundleInput struct {
	IdentityName string
	AccessToken  string
	ProviderName string
	CABundle     string
}

func RenderBundle(in BundleInput) (string, error) {
	if in.ProviderName == "" {
		in.ProviderName = ProviderName
	}
	funcs := template.FuncMap{
		"indent": func(spaces int, s string) string {
			pad := strings.Repeat(" ", spaces)
			trimmed := strings.TrimRight(s, "\n")
			return pad + strings.ReplaceAll(trimmed, "\n", "\n"+pad)
		},
	}
	t, err := template.New("bundle").Funcs(funcs).Parse(bundleTpl)
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	if err := t.Execute(&b, in); err != nil {
		return "", err
	}
	return b.String(), nil
}
