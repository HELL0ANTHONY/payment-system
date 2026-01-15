package models

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
)

type CreatePaymentRequest struct {
	UserID      string          `json:"user_id"`
	ServiceID   string          `json:"service_id"`
	Amount      decimal.Decimal `json:"amount"`
	Currency    string          `json:"currency"`
	Description string          `json:"description"`
}

func (r *CreatePaymentRequest) Validate() error {
	if r.UserID == "" {
		return ErrValidation("user_id is required")
	}

	if r.Amount.LessThanOrEqual(decimal.Zero) {
		return ErrValidation("amount must be positive")
	}

	if r.Currency == "" {
		return ErrValidation("currency is required")
	}

	return nil
}

type Response struct {
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
	Success bool   `json:"success"`
}

type PaymentDTO struct {
	CreatedAt   time.Time `json:"created_at"`
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	ServiceID   string    `json:"service_id"`
	Amount      string    `json:"amount"`
	Currency    string    `json:"currency"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
}

func SuccessJSON(data any) string {
	b, _ := json.Marshal(Response{Success: true, Data: data})

	return string(b)
}

func ErrorJSON(msg string) string {
	b, _ := json.Marshal(Response{Success: false, Error: msg})

	return string(b)
}

func Headers() map[string]string {
	return map[string]string{
		"Content-Type":                 "application/json",
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "GET, POST, OPTIONS",
	}
}

type ValidationError struct{ msg string }

func (e ValidationError) Error() string { return e.msg }

func ErrValidation(msg string) error { return ValidationError{msg} }
