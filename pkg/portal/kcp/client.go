package kcp

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	IdentityGVR = schema.GroupVersionResource{Group: "access.platform-mesh.io", Version: "v1alpha1", Resource: "accessidentities"}
	AccessGVR   = schema.GroupVersionResource{Group: "access.platform-mesh.io", Version: "v1alpha1", Resource: "accesses"}

	LabelPortalUser = "wildwest.dev/portal-user"
	AnnotationNote  = "wildwest.dev/note"
)

type Client struct {
	Dyn       dynamic.Interface
	Core      kubernetes.Interface
	Namespace string
}

func New(kubeconfigPath, namespace string) (*Client, error) {
	cfg, err := loadConfig(kubeconfigPath)
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

func loadConfig(kubeconfigPath string) (*rest.Config, error) {
	if kubeconfigPath != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}
	return rest.InClusterConfig()
}
