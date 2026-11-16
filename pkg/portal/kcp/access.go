package kcp

import (
	"context"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func (c *Client) GetAccessRaw(ctx context.Context, name string) (*unstructured.Unstructured, error) {
	return c.Dyn.Resource(AccessGVR).Namespace(c.Namespace).Get(ctx, name, metav1.GetOptions{})
}

type Access struct {
	Name             string
	IdentityName     string
	Phase            string
	LastRefreshed    time.Time
	KubeconfigSecret string
	CreatedAt        time.Time
}

func (c *Client) ListAccesses(ctx context.Context) ([]*Access, error) {
	list, err := c.Dyn.Resource(AccessGVR).Namespace(c.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	out := make([]*Access, 0, len(list.Items))
	for i := range list.Items {
		out = append(out, accessFromUnstructured(&list.Items[i]))
	}
	return out, nil
}

func (c *Client) GetAccess(ctx context.Context, name string) (*Access, error) {
	u, err := c.Dyn.Resource(AccessGVR).Namespace(c.Namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return accessFromUnstructured(u), nil
}

func (c *Client) DeleteAccess(ctx context.Context, name string) error {
	err := c.Dyn.Resource(AccessGVR).Namespace(c.Namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func accessFromUnstructured(u *unstructured.Unstructured) *Access {
	identityName, _, _ := unstructured.NestedString(u.Object, "spec", "identityRef", "name")
	kubeconfigSecret, _, _ := unstructured.NestedString(u.Object, "spec", "kubeconfigSecretRef", "name")
	phase, _, _ := unstructured.NestedString(u.Object, "status", "phase")
	lastRefreshedStr, _, _ := unstructured.NestedString(u.Object, "status", "lastRefreshed")
	var lastRefreshed time.Time
	if lastRefreshedStr != "" {
		lastRefreshed, _ = time.Parse(time.RFC3339, lastRefreshedStr)
	}
	return &Access{
		Name:             u.GetName(),
		IdentityName:     identityName,
		Phase:            phase,
		LastRefreshed:    lastRefreshed,
		KubeconfigSecret: kubeconfigSecret,
		CreatedAt:        u.GetCreationTimestamp().Time,
	}
}
