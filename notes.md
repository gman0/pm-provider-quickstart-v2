(workspce) Bind providers APIExport:

```
kubectl kcp bind apiexport root:platform-mesh-system:providers.platform-mesh.io --accept-all-permission-claims
```

Create the provider cluster:

```
echo '
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
 - role: control-plane
   extraPortMappings:
     - containerPort: 32000
       hostPort: 32000
       listenAddress: "127.0.0.1"
       protocol: TCP
'  | kind-with-registry.sh wildwest-provider
kind get kubeconfig -n wildwest-provider > kubeconfig
set KUBECONFIG ./kubeconfig
```

(provider) coredns setup:

```yaml
  Corefile: |
    .:53 {
        errors
        health {
           lameduck 5s
        }
        ready
        kubernetes cluster.local in-addr.arpa ip6.arpa {
           pods insecure
           fallthrough in-addr.arpa ip6.arpa
           ttl 30
        }
        prometheus :9153
        template IN A pm.127-0-0-1.sslip.io {
           match (^|\.)pm\.127-0-0-1\.sslip\.io\.$
           answer "{{ .Name }} 30 IN A 10.89.0.32"
           fallthrough
        }
        forward . /etc/resolv.conf {
           max_concurrent 1000
        }
        cache 30 {
           disable success cluster.local
           disable denial cluster.local
        }
        loop
        reload
        loadbalance
    }
```

Deploy cert-manager:

```
helm install \
  cert-manager oci://quay.io/jetstack/charts/cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --set crds.enabled=true
```

Build pm-access:

```
podman build -t localhost:5001/platform-mesh-access -f ~/ws/platform-mesh/sdk/access/Dockerfile ~/ws/platform-mesh
podman push localhost:5001/platform-mesh-access
```

Create selfsigned issuer:
```
echo '
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: selfsigned-cluster-issuer
spec:
  selfSigned: {}
' | kubectl apply -f -
```

Deploy pm-access:

```
NODE_IP=(podman inspect wildwest-provider-control-plane --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}') \
helm upgrade --install platform-mesh-access-sdk ~/ws/platform-mesh/charts/platform-mesh-access-sdk \
                                   --namespace wildwest \
                                   --set image.registry=localhost:5001 \
                                   --set image.repository=platform-mesh-access \
                                   --set image.tag=latest \
                                   --set image.pullPolicy=Always \
                                   --set publicUrl=https://$NODE_IP:32000 \
                                   --set service.type=NodePort \
                                   --set service.nodePort=32000 \
                                   --set tls.issuerRef=selfsigned-cluster-issuer \
                                   --set-json "tls.ipAddresses=[\"$NODE_IP\"]"
```

---

(provider cluster) Apply an AccessIdentity

```
kubectl-ns wildwest
echo '
kind: AccessIdentity
apiVersion: access-sdk.platform-mesh.io/v1alpha1
metadata:
  name: bob
spec: {}
' | kubectl apply -f -
```

(provider cluster) Get the access token

```
kubectl get secret bob-access-token -o json | jq .data.accessToken -r | base64 -d | wl-copy
```

(workspce) Prepare the provider integration:

```
kubectl create secret generic wildwest-token \
  --from-literal accessToken=pma1.aHR0cHM6Ly8xMC44OS4wLjMzOjMyMDAw.-jsquSTI7j8GQdoU_kNjkQ.TDxDwzES81ztNQ4lHddgRQOMjfTnwo7rw3o2sTkLnSI
```

(provider cluster) get the CA bundle:
```
kubectl get secret platform-mesh-access-tls -o json | jq '.data."ca.crt"' -r | base64 -d | wl-copy
```

```
echo '
kind: Provider
apiVersion: providers.platform-mesh.io/v1alpha1
metadata:
  name: wildwest
spec:
  remote:
    caBundle: |
      -----BEGIN CERTIFICATE-----
      MIIC8TCCAdmgAwIBAgIUVJocQHx4VOo1Nt5WQB7JazXkq/swDQYJKoZIhvcNAQEL
      BQAwADAeFw0yNjExMTIxMzA3MjBaFw0yNzAyMTAxMzA3MjBaMAAwggEiMA0GCSqG
      SIb3DQEBAQUAA4IBDwAwggEKAoIBAQC45/piwpmQq+A390qLZoPUtKi/XBayr9Wl
      AVf8FZEXJgPxd3kjEqSJRvgpVIRYEgfrvQTkvWhfkLFEUmuK7SmgnfNzPV9imTQP
      1kHVFYEelALKsn3fgo93jNSurIodXzZMovaUMOaDaT+9aGNg6/8tBS6QwMrPxgr4
      MMd2tLSO2H7RLV8xQ0qlIOkmJJ6XI4taS6zYVJyHJp56UYqm7q3pmpetR5Jh1EdM
      pc7AYPxmS2Nd8IBQ3DaYwUzFj0p5Xlvz1YsRMwZojEJPbmgpGd+00IGir8Zeoz6H
      1mYOQWjBtULwRMF+BaSztimKqMGs9e/42d8D9CD6Wd/ZQOEYv2JhAgMBAAGjYzBh
      MA4GA1UdDwEB/wQEAwIFoDAMBgNVHRMBAf8EAjAAMEEGA1UdEQEB/wQ3MDWCLXBs
      YXRmb3JtLW1lc2gtYWNjZXNzLnBsYXRmb3JtLW1lc2gtc3lzdGVtLnN2Y4cEClkA
      ITANBgkqhkiG9w0BAQsFAAOCAQEAGOsbdL7dj+7G27wt8Gufr0sOaXqyIyh6dd5u
      eCxxEwZsbvneTWPjY/B13TZMuWjMXHL7xP0vUYMWrVd8AUjw7NshqbrIGJQuMqEF
      SLaC7e8dTZusiZ6TEcukLUYfo8oAbPPdKfKOiyIn+UYhKdOK2DoGmmXc4o3BU3KW
      6ADqembmPqAPHtQ7D1gh+C+7VzQGDZWegKaVAxzrYA3bFY7aSSSbruFekeTu0lWf
      MkHkUgH8bcfDJgh2XTexYhhH7b6hA8vvw2/QAzu4HDJRrSCvjCjaCltp17hm8Q4A
      sxkmw652qQYsYBQHvJKhjvJyysWz3YvGAzS3nOc4npm5DpFugg==
      -----END CERTIFICATE-----

' | kubectl apply -f -
```

```
echo '
kind: ProviderAccessRequest
apiVersion: providers.platform-mesh.io/v1alpha1
metadata:
  name: wildwest
spec:
  accessTokenSecretRef:
    name: wildwest-token
    namespace: default
  providerRef:
    name: wildwest
' | kubectl apply -f -
```

Build api-syncagent-deployer

```
sh -c 'cd ~/ws/kcp-api-syncagent-deployer && make fullbuild'
```

(provider) Deploy flux

```
helm upgrade --install flux oci://ghcr.io/fluxcd-community/charts/flux2 \
  --namespace flux-system \
  --create-namespace \
  --set sourceController.create=true \
  --set helmController.create=true \
  --set kustomizeController.create=false \
  --set notificationController.create=false \
  --set imageAutomationController.create=false \
  --set imageReflectionController.create=false
```

(provider) Deploy api-syncagent-deployer

```
helm \
  upgrade --install -n wildwest --create-namespace \
  apisyncagent-deployer ~/ws/kcp-api-syncagent-deployer/charts/kcp-api-syncagent-deployer \
  --set image.repository=localhost:5001/kcp-api-syncagent-deployer \
  --set image.tag=latest \
  --set image.pullPolicy=Always \
  --set apiExportEndpointSliceName=wildwest \
  --set secretLabelSelector=access-sdk.platform-mesh.io/kubeconfig=true
```

(provider) Build wildwest:

```
make image-push IMAGE_REGISTRY=localhost:5001 IMAGE_TAG=latest
```

```
helm \
  install -n wildwest --create-namespace \
  wildwest-bootstrapper ./deploy/helm/wildwest-bootstrapper \
  --set image.repository=localhost:5001/provider-quickstart \
  --set image.tag=latest \
  --set image.pullPolicy=Always \


helm \
  install -n wildwest --create-namespace \
  wildwest-controller ./deploy/helm/wildwest-controller \
  --set init.enabled=true \
  --set init.kubeconfig.secretName=bob-hcq1dw-kubeconfig \
  --set kubeconfig.secretName=bob-hcq1dw-kubeconfig

helm \
  install -n wildwest --create-namespace \
  wildwest-portal ./deploy/helm/wildwest-portal
```
