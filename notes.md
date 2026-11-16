# Building

Build pm-access:

```
podman build -t localhost:5001/platform-mesh-access -f ~/ws/platform-mesh/sdk/access/Dockerfile ~/ws/platform-mesh
podman push localhost:5001/platform-mesh-access
```

Build api-syncagent-deployer

```
sh -c 'cd ~/ws/kcp-api-syncagent-deployer && make fullbuild'
```

Build wildwest

```
sh -c 'cd ~/ws/pm-provider-quickstart-v2 && make image-push IMAGE_REGISTRY=localhost:5001 IMAGE_TAG=latest'
```

Build provider portal

```
sh -c 'cd ~/ws/pm-provider-quickstart-v2 && make portal-image-push IMAGE_REGISTRY=localhost:5001 IMAGE_TAG=latest'
```

# Cluster setup

Setup the provider cluster:

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

# cert-manager
helm install \
  cert-manager oci://quay.io/jetstack/charts/cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --set crds.enabled=true

# Network setup

helm install eg oci://docker.io/envoyproxy/gateway-helm \
  --version v1.10.0 -n envoy-gateway-system --create-namespace

echo "
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: selfsigned-cluster-issuer
  namespace: envoy-gateway-system
spec:
  selfSigned: {}
" | kubectl apply -f -

NODE_IP=(podman inspect wildwest-provider-control-plane --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}') \
echo "
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: shared-gateway-tls
  namespace: envoy-gateway-system
spec:
  secretName: shared-gateway-tls
  issuerRef:
    name: selfsigned-cluster-issuer
    kind: ClusterIssuer
    group: cert-manager.io
  ipAddresses:
  - 127.0.0.1
  - $NODE_IP
  dnsNames:
  - localhost
  - wildwest.127-0-0-1.sslip.io
" | kubectl apply -f -

echo '
apiVersion: gateway.envoyproxy.io/v1alpha1
kind: EnvoyProxy
metadata:
  name: shared-nodeport
  namespace: envoy-gateway-system
spec:
  provider:
    type: Kubernetes
    kubernetes:
      envoyService:
        type: NodePort
        patch:
          type: StrategicMerge
          value:
            spec:
              ports:
                - name: https-443
                  port: 443
                  nodePort: 32000
' | kubectl apply -f -

echo '
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: eg
spec:
  controllerName: gateway.envoyproxy.io/gatewayclass-controller
' | kubectl apply -f -

echo '
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: shared
  namespace: envoy-gateway-system
spec:
  gatewayClassName: eg
  infrastructure:
    parametersRef:
      group: gateway.envoyproxy.io
      kind: EnvoyProxy
      name: shared-nodeport
  listeners:
    - name: https
      protocol: HTTPS
      port: 443
      tls:
        mode: Terminate
        certificateRefs:
          - { group: "", kind: Secret, name: shared-gateway-tls }
      allowedRoutes:
        namespaces:
          from: All
' | kubectl apply -f -

```

(provider) coredns setup; need to get PM's node IP:

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
           answer "{{ .Name }} 30 IN A 10.89.0.36"
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

(provider) Deploy flux (needed by api-syncagent-deployer):

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

# Deploy

(provider) Deploy pm-access:

```
NODE_IP=(podman inspect wildwest-provider-control-plane --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}') \
  helm upgrade --install platform-mesh-access-sdk \
    ~/ws/platform-mesh/charts/platform-mesh-access-sdk \
    -n wildwest --create-namespace \
    --set image.registry=localhost:5001 \
    --set image.repository=platform-mesh-access \
    --set image.tag=latest --set image.pullPolicy=Always \
    --set tls.enabled=false \
    --set service.type=ClusterIP \
    --set publicUrl=https://$NODE_IP:32000/pm \
    --set httpRoute.enabled=true \
    --set httpRoute.pathPrefix=/pm \
    --set-json 'httpRoute.parentRefs=[{"group":"gateway.networking.k8s.io","kind":"Gateway","name":"shared","namespace":"envoy-gateway-system"}]'
```

(provider) Deploy api-syncagent-deployer

```
helm \
  upgrade --install -n wildwest --create-namespace \
  api-syncagent-deployer ~/ws/kcp-api-syncagent-deployer/charts/api-syncagent-deployer \
  --set image.repository=localhost:5001/api-syncagent-deployer \
  --set image.tag=latest \
  --set image.pullPolicy=Always \
  --set apiExportEndpointSliceName=cowboys.wildwest.dev \
  --set secretLabelSelector=access-sdk.platform-mesh.io/kubeconfig=true
```

(provider) Deploy wildwest bootstrapper and controller:

```
helm \
  upgrade --install -n wildwest --create-namespace \
  wildwest-bootstrapper ./deploy/helm/wildwest-bootstrapper \
  --set image.repository=localhost:5001/provider-quickstart-v2 \
  --set image.tag=latest \
  --set image.pullPolicy=Always \
  --set secretLabelSelector=access-sdk.platform-mesh.io/kubeconfig=true

helm \
  upgrade --install -n wildwest --create-namespace \
  wildwest-controller ./deploy/helm/wildwest-controller \
  --set image.repository=localhost:5001/provider-quickstart-v2 \
  --set image.tag=latest \
  --set image.pullPolicy=Always

helm \
  upgrade --install wildwest-portal \
  ~/ws/pm-provider-quickstart-v2/deploy/helm/wildwest-portal \
  -n wildwest --create-namespace \
  --set image.repository=localhost:5001/provider-quickstart-v2-portal \
  --set image.tag=latest --set image.pullPolicy=Always \
  --set env.namespace=wildwest \
  --set service.type=ClusterIP \
  --set httpRoute.enabled=true \
  --set-json 'httpRoute.parentRefs=[{"group":"gateway.networking.k8s.io","kind":"Gateway","name":"shared","namespace":"envoy-gateway-system"}]' \
  --set-json 'httpRoute.hostnames=["wildwest.127-0-0-1.sslip.io"]'

```

# Onboard the provider

(provider) Apply the PublishedResource template:

```
kubectl-ns wildwest
kubectl apply -f ./config/provider/cowboys.publishedresource.template.yaml
```

(provider) Apply an AccessIdentity

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

(provider) Get the access token

```
kubectl get secret bob-access-token -o json | jq .data.accessToken -r | base64 -d | wl-copy
```

(workspce) Bind providers APIExport:

```
kubectl kcp bind apiexport root:platform-mesh-system:providers.platform-mesh.io --accept-all-permission-claims
```

(workspce) Apply provider's access token:

```
kubectl create secret generic wildwest-token \
  --from-literal accessToken=pma1.aHR0cHM6Ly8xMC44OS4wLjM3OjMyMDAwL3Bt.NunmQTcelNQYxp2RYwpITA.FnJdzEo86RtYyT85K3-loYQkyYn9OfiRVAZZb1qYeGo
```

(provider) get the CA bundle:
```
kubectl -n envoy-gateway-system get secret shared-gateway-tls -o json | jq '.data."ca.crt"' -r | base64 -d | wl-copy
```

(workspce) Create Provider:
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
      MIIC8DCCAdigAwIBAgIUAsyajAEAQxIv8MzfH/dilPh4OqMwDQYJKoZIhvcNAQEL
      BQAwADAeFw0yNjExMTYwOTM3NDlaFw0yNzAyMTQwOTM3NDlaMAAwggEiMA0GCSqG
      SIb3DQEBAQUAA4IBDwAwggEKAoIBAQCl58FnF+b/zzD9QP0mfzuaGFEjr0br/0NC
      yN5Yu6pYE9pvC/HemIMjfuUkY/sQg8BXUAZBKi/OpI028oQmSsLre9Dno8w9BhTD
      8QDIZuFCA7VpXkRXwMDXYKE30/5csU4TkoqP+Sk93OBJozA/RLDH8uj0VNVLVTr0
      aGdpOj22N42Eyr9tbLXyDbeMf6KxhWRHOYI6k6boF8k0gJimf5AETtiSiRTSCjjH
      5MmtBeDVhtVvoDK/kxKnrk2oK29ZjirUKm0LFZHEtkRhH9HjbuZ2flZHFjfSbt/4
      7mAQS+v7BfmBoVQiu+/+bfMYZubZV0goM3foYmWH26CzfIqQl+MPAgMBAAGjYjBg
      MA4GA1UdDwEB/wQEAwIFoDAMBgNVHRMBAf8EAjAAMEAGA1UdEQEB/wQ2MDSCCWxv
      Y2FsaG9zdIIbd2lsZHdlc3QuMTI3LTAtMC0xLnNzbGlwLmlvhwR/AAABhwQKWQAl
      MA0GCSqGSIb3DQEBCwUAA4IBAQAA02mjA6cdOoMXTd1kudFwZ/DAMQ2KaqzT7SDL
      JHFYiw0yjpg7CdDac7n+St/IVkvjDq3iXXBNMBCIlgsrMoq5H/7newRDTnCHUshB
      69auO/DzzXcf4ciBkVgljvuhtiWapqTVC+MSuTtwpQa2NRF5tKQgivEfmuPJfoqT
      r03cvzER95boruK25f1pn+9tq7PqWy96i3HYUAHL3WZ4ATizlMeF/5rRu9AumFUZ
      FL5kJYNAfZ1yv7rWOBqsMfplnX+A1QGar6nDXO+O/upSTG66VvufgdgCLaRqC2Uw
      y6NQeS1QK/AD42VftplcCskiQ9q3BNhKndNtEn3FmErRKEmz
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




---

```
kubectl -s 'https://kcp.api.pm.127-0-0-1.sslip.io:31000/services/marketplace/clusters/24735is03e2i0dvd' get marketplaceentries -o yaml
```
