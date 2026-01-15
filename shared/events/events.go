package events

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Event types.
const (
	PaymentInitiated       = "payment.initiated"
	PaymentCompleted       = "payment.completed"
	PaymentFailed          = "payment.failed"
	FundsReserved          = "wallet.funds_reserved"
	FundsReservationFailed = "wallet.reservation_failed"
	FundsDeducted          = "wallet.funds_deducted"
	FundsReleased          = "wallet.funds_released"
	GatewayPaymentApproved = "gateway.payment_approved"
	GatewayPaymentRejected = "gateway.payment_rejected"
)

// Event is the base structure for all events.
type Event struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	OccurredAt    time.Time       `json:"occurred_at"`
	PaymentID     string          `json:"payment_id"`
	UserID        string          `json:"user_id"`
	Amount        decimal.Decimal `json:"amount"`
	Currency      string          `json:"currency,omitempty"`
	Reason        string          `json:"reason,omitempty"`
	ReservationID string          `json:"reservation_id,omitempty"`
	GatewayRef    string          `json:"gateway_ref,omitempty"`
}

// New creates a new event with common fields.
func New(eventType, paymentID, userID string) Event {
	return Event{
		ID:         uuid.New().String(),
		Type:       eventType,
		OccurredAt: time.Now().UTC(),
		PaymentID:  paymentID,
		UserID:     userID,
	}
}

// WithAmount adds amount info to the event.
func (e *Event) WithAmount(amount decimal.Decimal, currency string) *Event {
	e.Amount = amount
	e.Currency = currency

	return e
}

// WithReason adds a reason to the event.
func (e *Event) WithReason(reason string) *Event {
	e.Reason = reason

	return e
}

// WithReservation adds reservation info to the event.
func (e *Event) WithReservation(reservationID string) *Event {
	e.ReservationID = reservationID

	return e
}

// WithGatewayRef adds gateway reference to the event.
func (e *Event) WithGatewayRef(ref string) *Event {
	e.GatewayRef = ref

	return e
}
