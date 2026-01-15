package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/HELL0ANTHONY/payment-system/shared/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mocks

type mockDB struct {
	mock.Mock
}

func (m *mockDB) PutItem(
	ctx context.Context,
	input *dynamodb.PutItemInput,
	opts ...func(*dynamodb.Options),
) (*dynamodb.PutItemOutput, error) {
	args := m.Called(ctx, input)
	return &dynamodb.PutItemOutput{}, args.Error(1)
}

type mockPublisher struct {
	mock.Mock
}

func (m *mockPublisher) Publish(ctx context.Context, queueURL string, event *events.Event) error {
	args := m.Called(ctx, queueURL, event)
	return args.Error(0)
}

// Tests

func TestHandleFailedEvent_Retry(t *testing.T) {
	ctx := context.Background()
	db := new(mockDB)
	pub := new(mockPublisher)

	pub.On("Publish", ctx, "http://wallet-queue", mock.Anything).Return(nil)

	svc := New(db, pub, "failed-events", "http://wallet-queue", 3)

	event := events.New(events.PaymentInitiated, "pay-123", "user-456")
	event.WithAmount(decimal.NewFromInt(100), "USD")
	body, _ := json.Marshal(event)

	err := svc.HandleFailedEvent(ctx, "msg-123", string(body), "wallet-queue-dlq", 1)

	assert.NoError(t, err)
	pub.AssertExpectations(t)
	db.AssertNotCalled(t, "PutItem") // Should retry, not store
}

func TestHandleFailedEvent_MaxRetriesExceeded(t *testing.T) {
	ctx := context.Background()
	db := new(mockDB)
	pub := new(mockPublisher)

	db.On("PutItem", ctx, mock.Anything).Return(nil, nil)

	svc := New(db, pub, "failed-events", "http://wallet-queue", 3)

	event := events.New(events.PaymentInitiated, "pay-123", "user-456")
	body, _ := json.Marshal(event)

	err := svc.HandleFailedEvent(ctx, "msg-123", string(body), "wallet-queue-dlq", 5)

	assert.NoError(t, err)
	db.AssertExpectations(t)
	pub.AssertNotCalled(t, "Publish") // Should store, not retry
}

func TestHandleFailedEvent_NonRetryable(t *testing.T) {
	ctx := context.Background()
	db := new(mockDB)
	pub := new(mockPublisher)

	db.On("PutItem", ctx, mock.Anything).Return(nil, nil)

	svc := New(db, pub, "failed-events", "http://wallet-queue", 3)

	// GatewayPaymentApproved is not retryable
	event := events.New(events.GatewayPaymentApproved, "pay-123", "user-456")
	body, _ := json.Marshal(event)

	err := svc.HandleFailedEvent(ctx, "msg-123", string(body), "gateway-queue-dlq", 1)

	assert.NoError(t, err)
	db.AssertExpectations(t)
	pub.AssertNotCalled(t, "Publish")
}

func TestHandleFailedEvent_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	db := new(mockDB)
	pub := new(mockPublisher)

	db.On("PutItem", ctx, mock.Anything).Return(nil, nil)

	svc := New(db, pub, "failed-events", "http://wallet-queue", 3)

	err := svc.HandleFailedEvent(ctx, "msg-123", "invalid json", "some-queue-dlq", 1)

	assert.NoError(t, err)
	db.AssertExpectations(t)
}

func TestIsRetryable(t *testing.T) {
	svc := New(nil, nil, "", "", 3)

	assert.True(t, svc.isRetryable(events.PaymentInitiated))
	assert.True(t, svc.isRetryable(events.FundsReserved))
	assert.False(t, svc.isRetryable(events.PaymentCompleted))
	assert.False(t, svc.isRetryable(events.GatewayPaymentApproved))
}
