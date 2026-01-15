package handler

import (
	"context"
	"log/slog"
	"strconv"

	awsEvents "github.com/aws/aws-lambda-go/events"

	"github.com/HELL0ANTHONY/payment-system/lambdas/error-handler/internal/service"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

// Handle processes DLQ messages.
func (h *Handler) Handle(ctx context.Context, sqsEvent awsEvents.SQSEvent) error {
	slog.Info("processing DLQ batch", "count", len(sqsEvent.Records))

	var lastErr error
	for _, record := range sqsEvent.Records {
		if err := h.processRecord(ctx, record); err != nil {
			slog.Error("failed to process DLQ record", "error", err, "message_id", record.MessageId)
			lastErr = err
		}
	}
	return lastErr
}

func (h *Handler) processRecord(ctx context.Context, record awsEvents.SQSMessage) error {
	source := h.getSource(record)
	retryCount := h.getRetryCount(record)

	return h.svc.HandleFailedEvent(ctx, record.MessageId, record.Body, source, retryCount)
}

func (h *Handler) getSource(record awsEvents.SQSMessage) string {
	if arn := record.EventSourceARN; arn != "" {
		return arn
	}
	return "unknown"
}

func (h *Handler) getRetryCount(record awsEvents.SQSMessage) int {
	if attr, ok := record.Attributes["ApproximateReceiveCount"]; ok {
		if count, err := strconv.Atoi(attr); err == nil {
			return count
		}
	}
	return 0
}
