package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/HELL0ANTHONY/payment-system/shared/events"
	awsEvents "github.com/aws/aws-lambda-go/events"

	"github.com/HELL0ANTHONY/payment-system/lambdas/metrics-collector/internal/service"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Handle(ctx context.Context, ebEvent *awsEvents.CloudWatchEvent) error {
	slog.Info("received event", "type", ebEvent.DetailType, "source", ebEvent.Source)

	var event events.Event
	if err := json.Unmarshal(ebEvent.Detail, &event); err != nil {
		return fmt.Errorf("unmarshal event: %w", err)
	}

	if err := h.svc.RecordEvent(ctx, &event); err != nil {
		slog.Error("failed to record event", "error", err)

		return err
	}

	return nil
}
