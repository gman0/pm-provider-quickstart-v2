package server

import (
	"net/http"

	"github.com/gman0/provider-quickstart-v2/pkg/portal/auth"
	"github.com/gman0/provider-quickstart-v2/pkg/portal/config"
	"github.com/gman0/provider-quickstart-v2/pkg/portal/db"
)

type Server struct {
	cfg   *config.Config
	store *db.Store
}

func New(cfg *config.Config, store *db.Store) *Server {
	return &Server{cfg: cfg, store: store}
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

	auth.New(s.store).Register(mux)

	mux.Handle("/", spaHandler())

	return mux
}
