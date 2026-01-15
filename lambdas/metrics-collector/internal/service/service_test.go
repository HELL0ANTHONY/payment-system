package service

import (
	"context"
	"testing"

	"github.com/HELL0ANTHONY/payment-system/shared/events"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mocks

type mockCloudWatch struct {
	mock.Mock
}

func (m *mockCloudWatch) PutMetricData(
	ctx context.Context,
	input *cloudwatch.PutMetricDataInput,
	opts ...func(*cloudwatch.Options),
) (*cloudwatch.PutMetricDataOutput, error) {
	args := m.Called(ctx, input)
	return &cloudwatch.PutMetricDataOutput{}, args.Error(1)
}

// Tests

func TestRecordEvent_PaymentInitiated(t *testing.T) {
	ctx := context.Background()
	cw := new(mockCloudWatch)

	cw.On("PutMetricData", ctx, mock.MatchedBy(func(input *cloudwatch.PutMetricDataInput) bool {
		return *input.Namespace == "PaymentSystem" && len(input.MetricData) == 2
	})).Return(nil, nil)

	svc := New(cw, "PaymentSystem")

	event := events.New(events.PaymentInitiated, "pay-123", "user-456")
	event.WithAmount(decimal.NewFromInt(100), "USD")

	err := svc.RecordEvent(ctx, &event)

	assert.NoError(t, err)
	cw.AssertExpectations(t)
}

func TestRecordEvent_PaymentCompleted(t *testing.T) {
	ctx := context.Background()
	cw := new(mockCloudWatch)

	cw.On("PutMetricData", ctx, mock.MatchedBy(func(input *cloudwatch.PutMetricDataInput) bool {
		// Should have: EventCount, PaymentAmount, PaymentSuccess
		return len(input.MetricData) == 3
	})).Return(nil, nil)

	svc := New(cw, "PaymentSystem")

	event := events.New(events.PaymentCompleted, "pay-123", "user-456")
	event.WithAmount(decimal.NewFromInt(100), "USD")

	err := svc.RecordEvent(ctx, &event)

	assert.NoError(t, err)
	cw.AssertExpectations(t)
}

func TestRecordEvent_PaymentFailed(t *testing.T) {
	ctx := context.Background()
	cw := new(mockCloudWatch)

	cw.On("PutMetricData", ctx, mock.MatchedBy(func(input *cloudwatch.PutMetricDataInput) bool {
		// Should have: EventCount, PaymentFailure (no amount)
		hasFailureMetric := false
		for _, m := range input.MetricData {
			if *m.MetricName == "PaymentFailure" {
				hasFailureMetric = true
			}
		}
		return hasFailureMetric
	})).Return(nil, nil)

	svc := New(cw, "PaymentSystem")

	event := events.New(events.PaymentFailed, "pay-123", "user-456")
	event.WithReason("insufficient funds")

	err := svc.RecordEvent(ctx, &event)

	assert.NoError(t, err)
	cw.AssertExpectations(t)
}

func TestRecordEvent_GatewayRejected(t *testing.T) {
	ctx := context.Background()
	cw := new(mockCloudWatch)

	cw.On("PutMetricData", ctx, mock.Anything).Return(nil, nil)

	svc := New(cw, "PaymentSystem")

	event := events.New(events.GatewayPaymentRejected, "pay-123", "user-456")
	event.WithReason("declined by issuer")

	err := svc.RecordEvent(ctx, &event)

	assert.NoError(t, err)
	cw.AssertExpectations(t)
}

func TestGetStats(t *testing.T) {
	svc := New(nil, "PaymentSystem")

	event := events.New(events.PaymentInitiated, "pay-123", "user-456")
	event.WithAmount(decimal.NewFromInt(250), "MXN")

	stats := svc.GetStats(context.Background(), &event)

	assert.Equal(t, "pay-123", stats["payment_id"])
	assert.Equal(t, "user-456", stats["user_id"])
	assert.Equal(t, "250", stats["amount"])
	assert.Equal(t, "MXN", stats["currency"])
}
