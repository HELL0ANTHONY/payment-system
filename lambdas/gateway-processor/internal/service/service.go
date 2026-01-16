package service

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/HELL0ANTHONY/payment-system/shared/events"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// EventPublisher defines the event publishing operations we need.
type EventPublisher interface {
	Publish(ctx context.Context, queueURL string, event *events.Event) error
}

// GatewayClient simulates external payment gateway.
type GatewayClient interface {
	ProcessPayment(
		ctx context.Context,
		amount decimal.Decimal,
		currency string,
	) (*GatewayResponse, error)
}

type GatewayResponse struct {
	Reference string
	ErrorCode string
	Message   string
	Approved  bool
}

type Service struct {
	publisher      EventPublisher
	gateway        GatewayClient
	walletQueueURL string
}

func New(pub EventPublisher, gateway GatewayClient, walletQueueURL string) *Service {
	return &Service{
		publisher:      pub,
		gateway:        gateway,
		walletQueueURL: walletQueueURL,
	}
}

func (s *Service) ProcessPayment(
	ctx context.Context,
	paymentID, userID, reservationID string,
	amount decimal.Decimal,
	currency string,
) error {
	slog.Info("processing payment with gateway", "payment_id", paymentID, "amount", amount.String())

	resp, err := s.gateway.ProcessPayment(ctx, amount, currency)
	if err != nil {
		slog.Error("gateway error", "error", err)

		return s.publishRejected(
			ctx,
			paymentID,
			userID,
			reservationID,
			"gateway_error",
			err.Error(),
		)
	}

	if !resp.Approved {
		slog.Warn("payment rejected by gateway", "code", resp.ErrorCode)

		return s.publishRejected(
			ctx,
			paymentID,
			userID,
			reservationID,
			resp.ErrorCode,
			resp.Message,
		)
	}

	return s.publishApproved(ctx, paymentID, userID, reservationID, resp.Reference)
}

func (s *Service) publishApproved(
	ctx context.Context,
	paymentID, userID, reservationID, gatewayRef string,
) error {
	event := events.New(events.GatewayPaymentApproved, paymentID, userID)
	event.WithReservation(reservationID).WithGatewayRef(gatewayRef)

	if err := s.publisher.Publish(ctx, s.walletQueueURL, &event); err != nil {
		return fmt.Errorf("publish approved event: %w", err)
	}

	slog.Info("payment approved", "payment_id", paymentID, "gateway_ref", gatewayRef)

	return nil
}

func (s *Service) publishRejected(
	ctx context.Context,
	paymentID, userID, reservationID, code, reason string,
) error {
	event := events.New(events.GatewayPaymentRejected, paymentID, userID)
	event.WithReservation(reservationID).WithReason(reason)

	if err := s.publisher.Publish(ctx, s.walletQueueURL, &event); err != nil {
		return fmt.Errorf("publish rejected event: %w", err)
	}

	slog.Warn("payment rejected", "payment_id", paymentID, "code", code)

	return nil
}

// MockGateway simulates an external payment gateway for testing.
type MockGateway struct {
	FailRate float64
}

func NewMockGateway(failRate float64) *MockGateway {
	return &MockGateway{FailRate: failRate}
}

func (g *MockGateway) ProcessPayment(
	_ context.Context,
	_ decimal.Decimal,
	_ string,
) (*GatewayResponse, error) {
	time.Sleep(time.Duration(50+rand.Intn(100)) * time.Millisecond)

	if rand.Float64() < g.FailRate {
		return &GatewayResponse{
			Approved:  false,
			ErrorCode: "DECLINED",
			Message:   "transaction declined by issuer",
		}, nil
	}

	return &GatewayResponse{
		Approved:  true,
		Reference: fmt.Sprintf("GW-%s", uuid.New().String()[:8]),
	}, nil
}
