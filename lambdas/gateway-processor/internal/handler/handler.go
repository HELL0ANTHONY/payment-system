package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/HELL0ANTHONY/payment-system/shared/events"
	awsEvents "github.com/aws/aws-lambda-go/events"

	"github.com/HELL0ANTHONY/payment-system/lambdas/gateway-processor/internal/service"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Handle(ctx context.Context, sqsEvent awsEvents.SQSEvent) error {
	slog.Info("processing batch", "count", len(sqsEvent.Records))

	var lastErr error

	for i := range sqsEvent.Records {
		record := sqsEvent.Records[i]
		if err := h.processRecord(ctx, &record); err != nil {
			slog.Error("failed to process record", "error", err, "message_id", record.MessageId)
			lastErr = err
		}
	}

	return lastErr
}

func (h *Handler) processRecord(ctx context.Context, record *awsEvents.SQSMessage) error {
	var event events.Event
	if err := json.Unmarshal([]byte(record.Body), &event); err != nil {
		return fmt.Errorf("unmarshal event: %w", err)
	}

	slog.Info("processing event", "type", event.Type, "payment_id", event.PaymentID)

	switch event.Type {
	case events.FundsReserved:
		return h.svc.ProcessPayment(
			ctx,
			event.PaymentID,
			event.UserID,
			event.ReservationID,
			event.Amount,
			event.Currency,
		)
	default:
		slog.Warn("unknown event type", "type", event.Type)
		return nil
	}
}
