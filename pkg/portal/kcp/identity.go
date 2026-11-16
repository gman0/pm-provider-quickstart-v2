package kcp

import (
	"context"
	"errors"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
)

var ErrTokenNotReady = errors.New("access token secret not populated yet")

type Identity struct {
	Name      string
	Note      string
	TokenID   string
	CreatedAt time.Time
}

func (c *Client) CreateIdentity(ctx context.Context, portalUserID, note string) (*Identity, error) {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(IdentityGVR.GroupVersion().WithKind("AccessIdentity"))
	u.SetNamespace(c.Namespace)
	u.SetGenerateName("wildwest-")
	u.SetLabels(map[string]string{LabelPortalUser: portalUserID})
	if note != "" {
		u.SetAnnotations(map[string]string{AnnotationNote: note})
	}
	_ = unstructured.SetNestedField(u.Object, "Always", "spec", "accept")

	created, err := c.Dyn.Resource(IdentityGVR).Namespace(c.Namespace).Create(ctx, u, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return identityFromUnstructured(created), nil
}

func (c *Client) GetIdentity(ctx context.Context, name string) (*Identity, error) {
	u, err := c.Dyn.Resource(IdentityGVR).Namespace(c.Namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return identityFromUnstructured(u), nil
}

func (c *Client) ListIdentities(ctx context.Context, portalUserID string) ([]*Identity, error) {
	sel := labels.SelectorFromSet(labels.Set{LabelPortalUser: portalUserID}).String()
	list, err := c.Dyn.Resource(IdentityGVR).Namespace(c.Namespace).List(ctx, metav1.ListOptions{LabelSelector: sel})
	if err != nil {
		return nil, err
	}
	out := make([]*Identity, 0, len(list.Items))
	for i := range list.Items {
		out = append(out, identityFromUnstructured(&list.Items[i]))
	}
	return out, nil
}

func (c *Client) DeleteIdentity(ctx context.Context, name string) error {
	err := c.Dyn.Resource(IdentityGVR).Namespace(c.Namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

// FetchCABundle reads the configured CA bundle Secret and returns the PEM
// payload. Returns "" if the ref isn't configured (caller should then omit
// spec.remote.caBundle from the Provider CR).
func (c *Client) FetchCABundle(ctx context.Context) (string, error) {
	if !c.CABundleRef.Valid() {
		return "", nil
	}
	sec, err := c.Core.CoreV1().Secrets(c.CABundleRef.Namespace).Get(ctx, c.CABundleRef.Name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	v, ok := sec.Data[c.CABundleRef.Key]
	if !ok || len(v) == 0 {
		return "", fmt.Errorf("key %q missing in secret %s/%s", c.CABundleRef.Key, c.CABundleRef.Namespace, c.CABundleRef.Name)
	}
	return string(v), nil
}

// FetchToken reads the plaintext pma1 token from the Secret owned by the
// AccessIdentity. Polls for up to the given timeout — the access SDK writes
// the Secret asynchronously after the identity is created.
func (c *Client) FetchToken(ctx context.Context, identityName string, timeout time.Duration) (string, error) {
	secretName := identityName + "-access-token"
	deadline := time.Now().Add(timeout)
	for {
		sec, err := c.Core.CoreV1().Secrets(c.Namespace).Get(ctx, secretName, metav1.GetOptions{})
		if err == nil {
			if v, ok := sec.Data["accessToken"]; ok && len(v) > 0 {
				return string(v), nil
			}
		} else if !apierrors.IsNotFound(err) {
			return "", err
		}
		if time.Now().After(deadline) {
			return "", fmt.Errorf("%w: %s", ErrTokenNotReady, secretName)
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
}

func identityFromUnstructured(u *unstructured.Unstructured) *Identity {
	note := ""
	if a := u.GetAnnotations(); a != nil {
		note = a[AnnotationNote]
	}
	tokenID, _, _ := unstructured.NestedString(u.Object, "status", "tokenID")
	return &Identity{
		Name:      u.GetName(),
		Note:      note,
		TokenID:   tokenID,
		CreatedAt: u.GetCreationTimestamp().Time,
	}
}
