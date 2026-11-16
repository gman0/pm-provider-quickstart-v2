package wildwest

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/gman0/provider-quickstart-v2/pkg/portal/auth"
	"github.com/gman0/provider-quickstart-v2/pkg/portal/db"
	"github.com/gman0/provider-quickstart-v2/pkg/portal/kcp"
)


type Handlers struct {
	Store *db.Store
	KCP   *kcp.Client
}

func New(store *db.Store, k *kcp.Client) *Handlers {
	return &Handlers{Store: store, KCP: k}
}

func (h *Handlers) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/tokens", auth.Required(h.Store, h.listTokens))
	mux.HandleFunc("POST /api/tokens", auth.Required(h.Store, h.createToken))
	mux.HandleFunc("GET /api/tokens/{identity}/bundle", auth.Required(h.Store, h.tokenBundle))
	mux.HandleFunc("DELETE /api/tokens/{identity}", auth.Required(h.Store, h.deleteToken))

	mux.HandleFunc("GET /api/accesses", auth.Required(h.Store, h.listAccesses))
	mux.HandleFunc("GET /api/accesses/{name}", auth.Required(h.Store, h.getAccess))
	mux.HandleFunc("GET /api/accesses/{name}/raw", auth.Required(h.Store, h.getAccessRaw))
	mux.HandleFunc("DELETE /api/accesses/{name}", auth.Required(h.Store, h.deleteAccess))
}

type tokenView struct {
	Identity    string    `json:"identity"`
	Note        string    `json:"note"`
	TokenID     string    `json:"tokenID"`
	CreatedAt   time.Time `json:"createdAt"`
	AccessCount int       `json:"accessCount"`
}

type createTokenRequest struct {
	Note string `json:"note"`
}

type createTokenResponse struct {
	Identity    string `json:"identity"`
	Note        string `json:"note"`
	AccessToken string `json:"accessToken"`
	Bundle      string `json:"bundle"`
}

type accessView struct {
	Name             string    `json:"name"`
	IdentityName     string    `json:"identityName"`
	Note             string    `json:"note,omitempty"`
	Phase            string    `json:"phase"`
	LastRefreshed    time.Time `json:"lastRefreshed,omitempty"`
	KubeconfigSecret string    `json:"kubeconfigSecret,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
}

func (h *Handlers) listTokens(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	ctx := r.Context()

	identities, err := h.KCP.ListIdentities(ctx, user.ID)
	if err != nil {
		slog.Error("list identities", "err", err)
		auth.WriteErr(w, http.StatusInternalServerError, "list failed")
		return
	}
	owned := make(map[string]bool, len(identities))
	for _, id := range identities {
		owned[id.Name] = true
	}

	accesses, err := h.KCP.ListAccesses(ctx)
	if err != nil {
		slog.Error("list accesses", "err", err)
		auth.WriteErr(w, http.StatusInternalServerError, "list accesses failed")
		return
	}
	counts := make(map[string]int, len(identities))
	for _, a := range accesses {
		if owned[a.IdentityName] {
			counts[a.IdentityName]++
		}
	}

	out := make([]tokenView, 0, len(identities))
	for _, id := range identities {
		out = append(out, tokenView{
			Identity:    id.Name,
			Note:        id.Note,
			TokenID:     id.TokenID,
			CreatedAt:   id.CreatedAt,
			AccessCount: counts[id.Name],
		})
	}
	auth.WriteJSON(w, http.StatusOK, out)
}

func (h *Handlers) createToken(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	var req createTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		auth.WriteErr(w, http.StatusBadRequest, "invalid json")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	id, err := h.KCP.CreateIdentity(ctx, user.ID, req.Note)
	if err != nil {
		slog.Error("create identity", "err", err)
		auth.WriteErr(w, http.StatusInternalServerError, "create failed")
		return
	}

	token, err := h.KCP.FetchToken(ctx, id.Name, 5*time.Second)
	if err != nil {
		slog.Error("fetch token", "err", err, "identity", id.Name)
		auth.WriteErr(w, http.StatusGatewayTimeout, "token secret not ready yet; retry")
		return
	}

	bundle, err := kcp.RenderBundle(kcp.BundleInput{IdentityName: id.Name, AccessToken: token})
	if err != nil {
		slog.Error("render bundle", "err", err)
		auth.WriteErr(w, http.StatusInternalServerError, "bundle failed")
		return
	}

	auth.WriteJSON(w, http.StatusCreated, createTokenResponse{
		Identity:    id.Name,
		Note:        id.Note,
		AccessToken: token,
		Bundle:      bundle,
	})
}

func (h *Handlers) tokenBundle(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	name := r.PathValue("identity")

	id, err := h.KCP.GetIdentity(r.Context(), name)
	if err != nil {
		h.k8sErr(w, err, "get identity")
		return
	}
	if !h.userOwns(r.Context(), user.ID, id.Name) {
		auth.WriteErr(w, http.StatusNotFound, "not found")
		return
	}

	token, err := h.KCP.FetchToken(r.Context(), name, 2*time.Second)
	if err != nil {
		slog.Error("fetch token", "err", err)
		auth.WriteErr(w, http.StatusGatewayTimeout, "token secret not readable")
		return
	}
	bundle, err := kcp.RenderBundle(kcp.BundleInput{IdentityName: name, AccessToken: token})
	if err != nil {
		auth.WriteErr(w, http.StatusInternalServerError, "bundle failed")
		return
	}
	auth.WriteJSON(w, http.StatusOK, createTokenResponse{
		Identity:    name,
		Note:        id.Note,
		AccessToken: token,
		Bundle:      bundle,
	})
}

func (h *Handlers) deleteToken(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	name := r.PathValue("identity")

	if !h.userOwns(r.Context(), user.ID, name) {
		auth.WriteErr(w, http.StatusNotFound, "not found")
		return
	}
	if err := h.KCP.DeleteIdentity(r.Context(), name); err != nil {
		slog.Error("delete identity", "err", err)
		auth.WriteErr(w, http.StatusInternalServerError, "delete failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) listAccesses(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	ctx := r.Context()

	identities, err := h.KCP.ListIdentities(ctx, user.ID)
	if err != nil {
		slog.Error("list identities", "err", err)
		auth.WriteErr(w, http.StatusInternalServerError, "list failed")
		return
	}
	noteByIdentity := make(map[string]string, len(identities))
	owned := make(map[string]bool, len(identities))
	for _, id := range identities {
		owned[id.Name] = true
		noteByIdentity[id.Name] = id.Note
	}

	accesses, err := h.KCP.ListAccesses(ctx)
	if err != nil {
		slog.Error("list accesses", "err", err)
		auth.WriteErr(w, http.StatusInternalServerError, "list accesses failed")
		return
	}
	out := make([]accessView, 0)
	for _, a := range accesses {
		if !owned[a.IdentityName] {
			continue
		}
		out = append(out, accessView{
			Name:             a.Name,
			IdentityName:     a.IdentityName,
			Note:             noteByIdentity[a.IdentityName],
			Phase:            a.Phase,
			LastRefreshed:    a.LastRefreshed,
			KubeconfigSecret: a.KubeconfigSecret,
			CreatedAt:        a.CreatedAt,
		})
	}
	auth.WriteJSON(w, http.StatusOK, out)
}

func (h *Handlers) getAccess(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	name := r.PathValue("name")

	a, err := h.KCP.GetAccess(r.Context(), name)
	if err != nil {
		h.k8sErr(w, err, "get access")
		return
	}
	// Resolve the owning identity note while checking ownership — same
	// label-selector list call either way, no extra round-trip cost.
	identities, err := h.KCP.ListIdentities(r.Context(), user.ID)
	if err != nil {
		slog.Error("list identities", "err", err)
		auth.WriteErr(w, http.StatusInternalServerError, "list failed")
		return
	}
	var note string
	owned := false
	for _, id := range identities {
		if id.Name == a.IdentityName {
			owned = true
			note = id.Note
			break
		}
	}
	if !owned {
		auth.WriteErr(w, http.StatusNotFound, "not found")
		return
	}
	auth.WriteJSON(w, http.StatusOK, accessView{
		Name:             a.Name,
		IdentityName:     a.IdentityName,
		Note:             note,
		Phase:            a.Phase,
		LastRefreshed:    a.LastRefreshed,
		KubeconfigSecret: a.KubeconfigSecret,
		CreatedAt:        a.CreatedAt,
	})
}

func (h *Handlers) getAccessRaw(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	name := r.PathValue("name")

	raw, err := h.KCP.GetAccessRaw(r.Context(), name)
	if err != nil {
		h.k8sErr(w, err, "get access raw")
		return
	}
	identityName, _, _ := unstructured.NestedString(raw.Object, "spec", "identityRef", "name")
	if !h.userOwns(r.Context(), user.ID, identityName) {
		auth.WriteErr(w, http.StatusNotFound, "not found")
		return
	}
	auth.WriteJSON(w, http.StatusOK, raw.Object)
}

func (h *Handlers) deleteAccess(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	name := r.PathValue("name")

	a, err := h.KCP.GetAccess(r.Context(), name)
	if err != nil {
		h.k8sErr(w, err, "get access")
		return
	}
	if !h.userOwns(r.Context(), user.ID, a.IdentityName) {
		auth.WriteErr(w, http.StatusNotFound, "not found")
		return
	}
	if err := h.KCP.DeleteAccess(r.Context(), name); err != nil {
		slog.Error("delete access", "err", err)
		auth.WriteErr(w, http.StatusInternalServerError, "delete failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// userOwns resolves the AccessIdentity named `name` and reports whether its
// portal-user label matches the caller. Used to gate every by-name endpoint.
func (h *Handlers) userOwns(ctx context.Context, userID, identityName string) bool {
	identities, err := h.KCP.ListIdentities(ctx, userID)
	if err != nil {
		return false
	}
	for _, id := range identities {
		if id.Name == identityName {
			return true
		}
	}
	return false
}

func (h *Handlers) k8sErr(w http.ResponseWriter, err error, op string) {
	if apierrors.IsNotFound(err) {
		auth.WriteErr(w, http.StatusNotFound, "not found")
		return
	}
	slog.Error(op, "err", err)
	auth.WriteErr(w, http.StatusInternalServerError, op+" failed")
}
