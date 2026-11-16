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
}

func New(store *db.Store) *Handlers {
	return &Handlers{Store: store}
}

func (h *Handlers) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/auth/register", h.register)
	mux.HandleFunc("POST /api/auth/login", h.login)
	mux.HandleFunc("POST /api/auth/logout", h.logout)
	mux.HandleFunc("GET /api/auth/me", h.required(h.me))
	mux.HandleFunc("DELETE /api/auth/me", h.required(h.deleteMe))
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
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if _, err := mail.ParseAddress(email); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid email")
		return
	}
	if len(req.Password) < 8 {
		writeErr(w, http.StatusBadRequest, "password too short (min 8)")
		return
	}
	hash, err := HashPassword(req.Password)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "hash failed")
		return
	}
	id := uuid.NewString()
	if err := h.Store.CreateUser(id, email, hash); err != nil {
		if errors.Is(err, db.ErrUserExists) {
			writeErr(w, http.StatusConflict, "email already registered")
			return
		}
		slog.Error("create user", "err", err)
		writeErr(w, http.StatusInternalServerError, "create failed")
		return
	}
	if err := h.startSession(w, id); err != nil {
		slog.Error("start session", "err", err)
		writeErr(w, http.StatusInternalServerError, "session failed")
		return
	}
	writeJSON(w, http.StatusCreated, meResponse{ID: id, Email: email, CreatedAt: time.Now()})
}

func (h *Handlers) login(w http.ResponseWriter, r *http.Request) {
	var req credsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	user, err := h.Store.GetUserByEmail(email)
	if err != nil {
		if errors.Is(err, db.ErrUserNotFound) {
			writeErr(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		slog.Error("get user", "err", err)
		writeErr(w, http.StatusInternalServerError, "login failed")
		return
	}
	if err := VerifyPassword(req.Password, user.PasswordHash); err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err := h.startSession(w, user.ID); err != nil {
		slog.Error("start session", "err", err)
		writeErr(w, http.StatusInternalServerError, "session failed")
		return
	}
	writeJSON(w, http.StatusOK, meResponse{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt})
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
	writeJSON(w, http.StatusOK, meResponse{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt})
}

func (h *Handlers) deleteMe(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	// §10: the portal should cascade-delete the user's AccessIdentities and
	// Accesses on the wildwest cluster before removing the DB row. The kcp
	// client is not wired yet (milestone 3); until then, just drop DB state.
	if err := h.Store.DeleteUser(user.ID); err != nil {
		slog.Error("delete user", "err", err)
		writeErr(w, http.StatusInternalServerError, "delete failed")
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

func (h *Handlers) required(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := CookieFromRequest(r)
		if id == "" {
			writeErr(w, http.StatusUnauthorized, "no session")
			return
		}
		sess, err := h.Store.GetSession(id)
		if err != nil || sess.ExpiresAt.Before(time.Now()) {
			ClearCookie(w)
			writeErr(w, http.StatusUnauthorized, "session expired")
			return
		}
		user, err := h.Store.GetUserByID(sess.UserID)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "user missing")
			return
		}
		if time.Until(sess.ExpiresAt) < SessionTTL-SessionRefresh {
			newExpires := time.Now().Add(SessionTTL)
			_ = h.Store.ExtendSession(id, newExpires)
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

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
