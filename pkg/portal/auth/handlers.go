package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/gman0/provider-quickstart-v2/pkg/portal/db"
)

type ctxKey int

const userCtxKey ctxKey = 1

type Handlers struct {
	Store *db.Store
	// BeforeDeleteUser, if set, runs before the DB row is removed on
	// DELETE /api/auth/me. Lets the server cascade-delete kcp-side state
	// (AccessIdentities / Accesses) before the user disappears.
	BeforeDeleteUser func(ctx context.Context, userID string) error
}

func New(store *db.Store) *Handlers {
	return &Handlers{Store: store}
}

func (h *Handlers) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/auth/register", h.register)
	mux.HandleFunc("POST /api/auth/login", h.login)
	mux.HandleFunc("POST /api/auth/logout", h.logout)
	mux.HandleFunc("GET /api/auth/me", Required(h.Store, h.me))
	mux.HandleFunc("DELETE /api/auth/me", Required(h.Store, h.deleteMe))
}

type credsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type meResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
}

func (h *Handlers) register(w http.ResponseWriter, r *http.Request) {
	var req credsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if _, err := mail.ParseAddress(email); err != nil {
		WriteErr(w, http.StatusBadRequest, "invalid email")
		return
	}
	if len(req.Password) < 8 {
		WriteErr(w, http.StatusBadRequest, "password too short (min 8)")
		return
	}
	hash, err := HashPassword(req.Password)
	if err != nil {
		WriteErr(w, http.StatusInternalServerError, "hash failed")
		return
	}
	id := uuid.NewString()
	if err := h.Store.CreateUser(id, email, hash); err != nil {
		if errors.Is(err, db.ErrUserExists) {
			WriteErr(w, http.StatusConflict, "email already registered")
			return
		}
		slog.Error("create user", "err", err)
		WriteErr(w, http.StatusInternalServerError, "create failed")
		return
	}
	if err := h.startSession(w, id); err != nil {
		slog.Error("start session", "err", err)
		WriteErr(w, http.StatusInternalServerError, "session failed")
		return
	}
	WriteJSON(w, http.StatusCreated, meResponse{ID: id, Email: email, CreatedAt: time.Now()})
}

func (h *Handlers) login(w http.ResponseWriter, r *http.Request) {
	var req credsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	user, err := h.Store.GetUserByEmail(email)
	if err != nil {
		if errors.Is(err, db.ErrUserNotFound) {
			WriteErr(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		slog.Error("get user", "err", err)
		WriteErr(w, http.StatusInternalServerError, "login failed")
		return
	}
	if err := VerifyPassword(req.Password, user.PasswordHash); err != nil {
		WriteErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err := h.startSession(w, user.ID); err != nil {
		slog.Error("start session", "err", err)
		WriteErr(w, http.StatusInternalServerError, "session failed")
		return
	}
	WriteJSON(w, http.StatusOK, meResponse{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt})
}

func (h *Handlers) logout(w http.ResponseWriter, r *http.Request) {
	if id := CookieFromRequest(r); id != "" {
		_ = h.Store.DeleteSession(id)
	}
	ClearCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) me(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	WriteJSON(w, http.StatusOK, meResponse{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt})
}

func (h *Handlers) deleteMe(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if h.BeforeDeleteUser != nil {
		if err := h.BeforeDeleteUser(r.Context(), user.ID); err != nil {
			slog.Error("cascade delete", "err", err)
			WriteErr(w, http.StatusInternalServerError, "cascade delete failed")
			return
		}
	}
	if err := h.Store.DeleteUser(user.ID); err != nil {
		slog.Error("delete user", "err", err)
		WriteErr(w, http.StatusInternalServerError, "delete failed")
		return
	}
	ClearCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) startSession(w http.ResponseWriter, userID string) error {
	id, err := NewSessionID()
	if err != nil {
		return err
	}
	expires := time.Now().Add(SessionTTL)
	if err := h.Store.CreateSession(id, userID, expires); err != nil {
		return err
	}
	SetCookie(w, id, expires)
	return nil
}

func Required(store *db.Store, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := CookieFromRequest(r)
		if id == "" {
			WriteErr(w, http.StatusUnauthorized, "no session")
			return
		}
		sess, err := store.GetSession(id)
		if err != nil || sess.ExpiresAt.Before(time.Now()) {
			ClearCookie(w)
			WriteErr(w, http.StatusUnauthorized, "session expired")
			return
		}
		user, err := store.GetUserByID(sess.UserID)
		if err != nil {
			WriteErr(w, http.StatusUnauthorized, "user missing")
			return
		}
		if time.Until(sess.ExpiresAt) < SessionTTL-SessionRefresh {
			newExpires := time.Now().Add(SessionTTL)
			_ = store.ExtendSession(id, newExpires)
			SetCookie(w, id, newExpires)
		}
		ctx := context.WithValue(r.Context(), userCtxKey, user)
		next(w, r.WithContext(ctx))
	}
}

func UserFromContext(ctx context.Context) *db.User {
	u, _ := ctx.Value(userCtxKey).(*db.User)
	return u
}

// WriteJSON and WriteErr are exported so other handler packages can use them.
func WriteJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

func WriteErr(w http.ResponseWriter, code int, msg string) {
	WriteJSON(w, code, map[string]string{"error": msg})
}
