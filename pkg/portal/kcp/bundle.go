package kcp

import (
	"bytes"
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
spec:
  remote: {}
---
apiVersion: providers.platform-mesh.io/v1alpha1
kind: ProviderAccessRequest
metadata:
  name: {{ .IdentityName }}-par
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
}

func RenderBundle(in BundleInput) (string, error) {
	t, err := template.New("bundle").Parse(bundleTpl)
	if err != nil {
		return "", err
	}
	if in.ProviderName == "" {
		in.ProviderName = ProviderName
	}
	var b bytes.Buffer
	if err := t.Execute(&b, in); err != nil {
		return "", err
	}
	return b.String(), nil
}
