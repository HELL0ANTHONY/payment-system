package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/HELL0ANTHONY/payment-system/shared/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

// CloudWatchClient defines the CloudWatch operations we need.
type CloudWatchClient interface {
	PutMetricData(
		ctx context.Context,
		params *cloudwatch.PutMetricDataInput,
		optFns ...func(*cloudwatch.Options),
	) (*cloudwatch.PutMetricDataOutput, error)
}

type Service struct {
	cw        CloudWatchClient
	namespace string
}

func New(cw CloudWatchClient, namespace string) *Service {
	return &Service{
		cw:        cw,
		namespace: namespace,
	}
}

func (s *Service) RecordEvent(ctx context.Context, event *events.Event) error {
	metrics := s.buildMetrics(event)
	if len(metrics) == 0 {
		return nil
	}

	_, err := s.cw.PutMetricData(ctx, &cloudwatch.PutMetricDataInput{
		Namespace:  aws.String(s.namespace),
		MetricData: metrics,
	})
	if err != nil {
		slog.Error("failed to put metrics", "error", err)

		return err
	}

	slog.Info("metrics recorded", "event_type", event.Type, "count", len(metrics))

	return nil
}

func (s *Service) buildMetrics(event *events.Event) []types.MetricDatum {
	now := time.Now()

	var metrics []types.MetricDatum

	metrics = append(metrics, types.MetricDatum{
		MetricName: aws.String("EventCount"),
		Value:      aws.Float64(1),
		Timestamp:  &now,
		Dimensions: []types.Dimension{
			{Name: aws.String("EventType"), Value: aws.String(event.Type)},
		},
		Unit: types.StandardUnitCount,
	})

	if !event.Amount.IsZero() {
		amount, _ := event.Amount.Float64()
		metrics = append(metrics, types.MetricDatum{
			MetricName: aws.String("PaymentAmount"),
			Value:      aws.Float64(amount),
			Timestamp:  &now,
			Dimensions: []types.Dimension{
				{Name: aws.String("EventType"), Value: aws.String(event.Type)},
				{Name: aws.String("Currency"), Value: aws.String(event.Currency)},
			},
			Unit: types.StandardUnitNone,
		})
	}

	switch event.Type {
	case events.PaymentCompleted:
		metrics = append(metrics, s.successMetric(now))
	case events.PaymentFailed, events.FundsReservationFailed, events.GatewayPaymentRejected:
		metrics = append(metrics, s.failureMetric(now, event.Type))
	}

	return metrics
}

func (s *Service) successMetric(t time.Time) types.MetricDatum {
	return types.MetricDatum{
		MetricName: aws.String("PaymentSuccess"),
		Value:      aws.Float64(1),
		Timestamp:  &t,
		Unit:       types.StandardUnitCount,
	}
}

func (s *Service) failureMetric(t time.Time, eventType string) types.MetricDatum {
	return types.MetricDatum{
		MetricName: aws.String("PaymentFailure"),
		Value:      aws.Float64(1),
		Timestamp:  &t,
		Dimensions: []types.Dimension{
			{Name: aws.String("FailureType"), Value: aws.String(eventType)},
		},
		Unit: types.StandardUnitCount,
	}
}

// GetStats returns aggregated stats (for testing/debugging).
func (s *Service) GetStats(_ context.Context, event *events.Event) map[string]any {
	return map[string]any{
		"event_type": event.Type,
		"payment_id": event.PaymentID,
		"user_id":    event.UserID,
		"amount":     event.Amount.String(),
		"currency":   event.Currency,
		"timestamp":  event.OccurredAt,
	}
}
