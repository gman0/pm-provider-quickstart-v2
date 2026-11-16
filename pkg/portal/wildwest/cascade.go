package wildwest

import (
	"context"

	"github.com/gman0/provider-quickstart-v2/pkg/portal/kcp"
)

// CascadeDeleteUser removes every Access and AccessIdentity owned by the
// given portal user. Deletes Accesses first (so the identity can't rematerialise
// them by racing a dangling handshake); then the identities.
func CascadeDeleteUser(ctx context.Context, k *kcp.Client, userID string) error {
	identities, err := k.ListIdentities(ctx, userID)
	if err != nil {
		return err
	}
	if len(identities) == 0 {
		return nil
	}
	owned := make(map[string]bool, len(identities))
	for _, id := range identities {
		owned[id.Name] = true
	}

	accesses, err := k.ListAccesses(ctx)
	if err != nil {
		return err
	}
	for _, a := range accesses {
		if !owned[a.IdentityName] {
			continue
		}
		if err := k.DeleteAccess(ctx, a.Name); err != nil {
			return err
		}
	}
	for _, id := range identities {
		if err := k.DeleteIdentity(ctx, id.Name); err != nil {
			return err
		}
	}
	return nil
}
