package server

import (
	"context"
	"net/http"

	"github.com/gman0/provider-quickstart-v2/pkg/portal/auth"
	"github.com/gman0/provider-quickstart-v2/pkg/portal/config"
	"github.com/gman0/provider-quickstart-v2/pkg/portal/db"
	"github.com/gman0/provider-quickstart-v2/pkg/portal/kcp"
	"github.com/gman0/provider-quickstart-v2/pkg/portal/profile"
	"github.com/gman0/provider-quickstart-v2/pkg/portal/wildwest"
)

type Server struct {
	cfg   *config.Config
	store *db.Store
	kcp   *kcp.Client
}

func New(cfg *config.Config, store *db.Store, k *kcp.Client) *Server {
	return &Server{cfg: cfg, store: store, kcp: k}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		if err := s.store.DB.Ping(); err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	authHandlers := auth.New(s.store)
	authHandlers.BeforeDeleteUser = func(ctx context.Context, userID string) error {
		return wildwest.CascadeDeleteUser(ctx, s.kcp, userID)
	}
	authHandlers.Register(mux)
	profile.New(s.store).Register(mux)
	wildwest.New(s.store, s.kcp).Register(mux)

	mux.Handle("/", spaHandler())

	return mux
}
