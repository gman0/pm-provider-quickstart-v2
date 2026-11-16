package kcp

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	IdentityGVR = schema.GroupVersionResource{Group: "access-sdk.platform-mesh.io", Version: "v1alpha1", Resource: "accessidentities"}
	AccessGVR   = schema.GroupVersionResource{Group: "access-sdk.platform-mesh.io", Version: "v1alpha1", Resource: "accesses"}

	LabelPortalUser = "wildwest.dev/portal-user"
	AnnotationNote  = "wildwest.dev/note"
)

type Client struct {
	Dyn       dynamic.Interface
	Core      kubernetes.Interface
	Namespace string

	CABundleRef SecretRef
}

type SecretRef struct {
	Namespace string
	Name      string
	Key       string
}

func (r SecretRef) Valid() bool { return r.Namespace != "" && r.Name != "" && r.Key != "" }

func New(namespace string) (*Client, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	core, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &Client{Dyn: dyn, Core: core, Namespace: namespace}, nil
}

// loadConfig prefers in-cluster config (the production path — portal runs in
// the provider's own cluster, same one that holds the AccessIdentity CRs) and
// falls back to the standard kubeconfig discovery (KUBECONFIG env or
// ~/.kube/config) so local `./bin/portal` works without extra setup.
func loadConfig() (*rest.Config, error) {
	if cfg, err := rest.InClusterConfig(); err == nil {
		return cfg, nil
	}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
}
