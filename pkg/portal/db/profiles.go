package db

import (
	"database/sql"
	"errors"
	"time"
)

type ContactProfile struct {
	FullName     string
	Company      string
	SupportEmail string
	UpdatedAt    time.Time
}

type PaymentProfile struct {
	CardHolder string
	CardLast4  string
	ExpMonth   int
	ExpYear    int
	UpdatedAt  time.Time
}

func (s *Store) GetContact(userID string) (*ContactProfile, error) {
	p := &ContactProfile{}
	err := s.DB.QueryRow(
		`SELECT full_name, company, support_email, updated_at FROM contact_profiles WHERE user_id = ?`,
		userID,
	).Scan(&p.FullName, &p.Company, &p.SupportEmail, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return &ContactProfile{}, nil
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Store) UpsertContact(userID string, p ContactProfile) error {
	_, err := s.DB.Exec(
		`INSERT INTO contact_profiles (user_id, full_name, company, support_email, updated_at)
		 VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(user_id) DO UPDATE SET
		   full_name = excluded.full_name,
		   company = excluded.company,
		   support_email = excluded.support_email,
		   updated_at = CURRENT_TIMESTAMP`,
		userID, p.FullName, p.Company, p.SupportEmail,
	)
	return err
}

func (s *Store) GetPayment(userID string) (*PaymentProfile, error) {
	p := &PaymentProfile{}
	err := s.DB.QueryRow(
		`SELECT card_holder, card_last4, exp_month, exp_year, updated_at FROM payment_profiles WHERE user_id = ?`,
		userID,
	).Scan(&p.CardHolder, &p.CardLast4, &p.ExpMonth, &p.ExpYear, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return &PaymentProfile{}, nil
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Store) UpsertPayment(userID string, p PaymentProfile) error {
	_, err := s.DB.Exec(
		`INSERT INTO payment_profiles (user_id, card_holder, card_last4, exp_month, exp_year, updated_at)
		 VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(user_id) DO UPDATE SET
		   card_holder = excluded.card_holder,
		   card_last4 = excluded.card_last4,
		   exp_month = excluded.exp_month,
		   exp_year = excluded.exp_year,
		   updated_at = CURRENT_TIMESTAMP`,
		userID, p.CardHolder, p.CardLast4, p.ExpMonth, p.ExpYear,
	)
	return err
}
