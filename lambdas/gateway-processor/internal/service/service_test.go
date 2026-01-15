package service

import (
	"context"
	"errors"
	"testing"

	"github.com/HELL0ANTHONY/payment-system/shared/events"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mocks

type mockPublisher struct {
	mock.Mock
}

func (m *mockPublisher) Publish(ctx context.Context, queueURL string, event *events.Event) error {
	args := m.Called(ctx, queueURL, event)
	return args.Error(0)
}

type mockGateway struct {
	mock.Mock
}

func (m *mockGateway) ProcessPayment(
	ctx context.Context,
	amount decimal.Decimal,
	currency string,
) (*GatewayResponse, error) {
	args := m.Called(ctx, amount, currency)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*GatewayResponse), args.Error(1)
}

// Tests

func TestProcessPayment_Approved(t *testing.T) {
	ctx := context.Background()
	pub := new(mockPublisher)
	gw := new(mockGateway)

	gw.On("ProcessPayment", ctx, decimal.NewFromInt(100), "USD").Return(&GatewayResponse{
		Approved:  true,
		Reference: "GW-12345",
	}, nil)
	pub.On("Publish", ctx, "http://wallet-queue", mock.Anything).Return(nil)

	svc := New(pub, gw, "http://wallet-queue")

	err := svc.ProcessPayment(ctx, "pay-123", "user-456", "res-789", decimal.NewFromInt(100), "USD")

	assert.NoError(t, err)
	gw.AssertExpectations(t)
	pub.AssertExpectations(t)

	// Verify the published event type
	publishCall := pub.Calls[0]
	event := publishCall.Arguments[2].(*events.Event)
	assert.Equal(t, events.GatewayPaymentApproved, event.Type)
	assert.Equal(t, "GW-12345", event.GatewayRef)
}

func TestProcessPayment_Rejected(t *testing.T) {
	ctx := context.Background()
	pub := new(mockPublisher)
	gw := new(mockGateway)

	gw.On("ProcessPayment", ctx, decimal.NewFromInt(100), "USD").Return(&GatewayResponse{
		Approved:  false,
		ErrorCode: "DECLINED",
		Message:   "insufficient funds at issuer",
	}, nil)
	pub.On("Publish", ctx, "http://wallet-queue", mock.Anything).Return(nil)

	svc := New(pub, gw, "http://wallet-queue")

	err := svc.ProcessPayment(ctx, "pay-123", "user-456", "res-789", decimal.NewFromInt(100), "USD")

	assert.NoError(t, err)

	// Verify the published event type
	publishCall := pub.Calls[0]
	event := publishCall.Arguments[2].(*events.Event)
	assert.Equal(t, events.GatewayPaymentRejected, event.Type)
	assert.Equal(t, "insufficient funds at issuer", event.Reason)
}

func TestProcessPayment_GatewayError(t *testing.T) {
	ctx := context.Background()
	pub := new(mockPublisher)
	gw := new(mockGateway)

	gw.On("ProcessPayment", ctx, decimal.NewFromInt(100), "USD").Return(nil, errors.New("timeout"))
	pub.On("Publish", ctx, "http://wallet-queue", mock.Anything).Return(nil)

	svc := New(pub, gw, "http://wallet-queue")

	err := svc.ProcessPayment(ctx, "pay-123", "user-456", "res-789", decimal.NewFromInt(100), "USD")

	assert.NoError(t, err) // Still publishes rejected event

	// Verify rejected event was published
	publishCall := pub.Calls[0]
	event := publishCall.Arguments[2].(*events.Event)
	assert.Equal(t, events.GatewayPaymentRejected, event.Type)
}

func TestMockGateway_AlwaysApproves(t *testing.T) {
	ctx := context.Background()
	gw := NewMockGateway(0.0) // 0% fail rate

	resp, err := gw.ProcessPayment(ctx, decimal.NewFromInt(100), "USD")

	assert.NoError(t, err)
	assert.True(t, resp.Approved)
	assert.NotEmpty(t, resp.Reference)
}

func TestMockGateway_AlwaysRejects(t *testing.T) {
	ctx := context.Background()
	gw := NewMockGateway(1.0) // 100% fail rate

	resp, err := gw.ProcessPayment(ctx, decimal.NewFromInt(100), "USD")

	assert.NoError(t, err)
	assert.False(t, resp.Approved)
	assert.Equal(t, "DECLINED", resp.ErrorCode)
}
