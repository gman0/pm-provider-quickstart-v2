package profile

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gman0/provider-quickstart-v2/pkg/portal/auth"
	"github.com/gman0/provider-quickstart-v2/pkg/portal/db"
)

type Handlers struct {
	Store *db.Store
}

func New(store *db.Store) *Handlers { return &Handlers{Store: store} }

func (h *Handlers) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/profile/contact", auth.Required(h.Store, h.getContact))
	mux.HandleFunc("PUT /api/profile/contact", auth.Required(h.Store, h.putContact))
	mux.HandleFunc("GET /api/profile/payment", auth.Required(h.Store, h.getPayment))
	mux.HandleFunc("PUT /api/profile/payment", auth.Required(h.Store, h.putPayment))
}

type contactDTO struct {
	FullName     string    `json:"fullName"`
	Company      string    `json:"company"`
	SupportEmail string    `json:"supportEmail"`
	UpdatedAt    time.Time `json:"updatedAt,omitempty"`
}

type paymentDTO struct {
	CardHolder string    `json:"cardHolder"`
	CardLast4  string    `json:"cardLast4"`
	ExpMonth   int       `json:"expMonth"`
	ExpYear    int       `json:"expYear"`
	UpdatedAt  time.Time `json:"updatedAt,omitempty"`
}

type paymentInputDTO struct {
	CardHolder string `json:"cardHolder"`
	CardNumber string `json:"cardNumber"`
	ExpMonth   int    `json:"expMonth"`
	ExpYear    int    `json:"expYear"`
}

func (h *Handlers) getContact(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	p, err := h.Store.GetContact(user.ID)
	if err != nil {
		slog.Error("get contact", "err", err)
		auth.WriteErr(w, http.StatusInternalServerError, "load failed")
		return
	}
	auth.WriteJSON(w, http.StatusOK, contactDTO{
		FullName: p.FullName, Company: p.Company, SupportEmail: p.SupportEmail, UpdatedAt: p.UpdatedAt,
	})
}

func (h *Handlers) putContact(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	var in contactDTO
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		auth.WriteErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	err := h.Store.UpsertContact(user.ID, db.ContactProfile{
		FullName: strings.TrimSpace(in.FullName), Company: strings.TrimSpace(in.Company),
		SupportEmail: strings.TrimSpace(in.SupportEmail),
	})
	if err != nil {
		slog.Error("upsert contact", "err", err)
		auth.WriteErr(w, http.StatusInternalServerError, "save failed")
		return
	}
	h.getContact(w, r)
}

func (h *Handlers) getPayment(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	p, err := h.Store.GetPayment(user.ID)
	if err != nil {
		slog.Error("get payment", "err", err)
		auth.WriteErr(w, http.StatusInternalServerError, "load failed")
		return
	}
	auth.WriteJSON(w, http.StatusOK, paymentDTO{
		CardHolder: p.CardHolder, CardLast4: p.CardLast4,
		ExpMonth: p.ExpMonth, ExpYear: p.ExpYear, UpdatedAt: p.UpdatedAt,
	})
}

func (h *Handlers) putPayment(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	var in paymentInputDTO
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		auth.WriteErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	last4 := extractLast4(in.CardNumber)
	if last4 == "" {
		auth.WriteErr(w, http.StatusBadRequest, "need at least 4 digits in card number")
		return
	}
	if in.ExpMonth < 1 || in.ExpMonth > 12 {
		auth.WriteErr(w, http.StatusBadRequest, "exp month 1-12")
		return
	}
	if in.ExpYear < 2000 || in.ExpYear > 2100 {
		auth.WriteErr(w, http.StatusBadRequest, "exp year out of range")
		return
	}
	err := h.Store.UpsertPayment(user.ID, db.PaymentProfile{
		CardHolder: strings.TrimSpace(in.CardHolder), CardLast4: last4,
		ExpMonth: in.ExpMonth, ExpYear: in.ExpYear,
	})
	if err != nil {
		slog.Error("upsert payment", "err", err)
		auth.WriteErr(w, http.StatusInternalServerError, "save failed")
		return
	}
	h.getPayment(w, r)
}

// extractLast4 returns the final 4 digits of a card number, ignoring spaces
// and dashes. Full PAN is never persisted — only these 4 digits make it to
// the DB. Returns "" if fewer than 4 digits are present.
func extractLast4(raw string) string {
	var digits strings.Builder
	for _, c := range raw {
		if c >= '0' && c <= '9' {
			digits.WriteRune(c)
		}
	}
	s := digits.String()
	if len(s) < 4 {
		return ""
	}
	return s[len(s)-4:]
}
