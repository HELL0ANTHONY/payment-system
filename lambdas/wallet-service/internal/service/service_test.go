package service

import (
	"context"
	"testing"

	"github.com/HELL0ANTHONY/payment-system/shared/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
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
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dynamodb.PutItemOutput), args.Error(1)
}

func (m *mockDB) GetItem(
	ctx context.Context,
	input *dynamodb.GetItemInput,
	opts ...func(*dynamodb.Options),
) (*dynamodb.GetItemOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*dynamodb.GetItemOutput), args.Error(1)
}

func (m *mockDB) UpdateItem(
	ctx context.Context,
	input *dynamodb.UpdateItemInput,
	opts ...func(*dynamodb.Options),
) (*dynamodb.UpdateItemOutput, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dynamodb.UpdateItemOutput), args.Error(1)
}

func (m *mockDB) Query(
	ctx context.Context,
	input *dynamodb.QueryInput,
	opts ...func(*dynamodb.Options),
) (*dynamodb.QueryOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*dynamodb.QueryOutput), args.Error(1)
}

type mockPublisher struct {
	mock.Mock
}

func (m *mockPublisher) Publish(ctx context.Context, queueURL string, event *events.Event) error {
	args := m.Called(ctx, queueURL, event)
	return args.Error(0)
}

// Tests

func TestReserveFunds_Success(t *testing.T) {
	ctx := context.Background()
	db := new(mockDB)
	pub := new(mockPublisher)

	walletItem := map[string]types.AttributeValue{
		"id":       &types.AttributeValueMemberS{Value: "wallet-123"},
		"user_id":  &types.AttributeValueMemberS{Value: "user-456"},
		"balance":  &types.AttributeValueMemberS{Value: "500"},
		"currency": &types.AttributeValueMemberS{Value: "USD"},
		"version":  &types.AttributeValueMemberN{Value: "1"},
	}

	db.On("Query", ctx, mock.Anything).Return(&dynamodb.QueryOutput{
		Items: []map[string]types.AttributeValue{walletItem},
	}, nil)
	db.On("PutItem", ctx, mock.Anything).Return(&dynamodb.PutItemOutput{}, nil)
	pub.On("Publish", ctx, "http://gateway-queue", mock.Anything).Return(nil)

	svc := New(db, pub, "wallets", "reservations", "http://gateway-queue")

	err := svc.ReserveFunds(ctx, "pay-789", "user-456", decimal.NewFromInt(100), "USD")

	assert.NoError(t, err)
	db.AssertExpectations(t)
	pub.AssertExpectations(t)
}

func TestReserveFunds_InsufficientFunds(t *testing.T) {
	ctx := context.Background()
	db := new(mockDB)
	pub := new(mockPublisher)

	walletItem := map[string]types.AttributeValue{
		"id":       &types.AttributeValueMemberS{Value: "wallet-123"},
		"user_id":  &types.AttributeValueMemberS{Value: "user-456"},
		"balance":  &types.AttributeValueMemberS{Value: "50"},
		"currency": &types.AttributeValueMemberS{Value: "USD"},
	}

	db.On("Query", ctx, mock.Anything).Return(&dynamodb.QueryOutput{
		Items: []map[string]types.AttributeValue{walletItem},
	}, nil)

	svc := New(db, pub, "wallets", "reservations", "http://gateway-queue")

	err := svc.ReserveFunds(ctx, "pay-789", "user-456", decimal.NewFromInt(100), "USD")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient funds")
}

func TestReserveFunds_WalletNotFound(t *testing.T) {
	ctx := context.Background()
	db := new(mockDB)
	pub := new(mockPublisher)

	db.On("Query", ctx, mock.Anything).Return(&dynamodb.QueryOutput{
		Items: []map[string]types.AttributeValue{},
	}, nil)

	svc := New(db, pub, "wallets", "reservations", "http://gateway-queue")

	err := svc.ReserveFunds(ctx, "pay-789", "unknown-user", decimal.NewFromInt(100), "USD")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "wallet not found")
}

func TestConfirmDeduction_Success(t *testing.T) {
	ctx := context.Background()
	db := new(mockDB)

	resItem := map[string]types.AttributeValue{
		"id":         &types.AttributeValueMemberS{Value: "res-123"},
		"payment_id": &types.AttributeValueMemberS{Value: "pay-456"},
		"user_id":    &types.AttributeValueMemberS{Value: "user-789"},
		"amount":     &types.AttributeValueMemberS{Value: "100"},
		"status":     &types.AttributeValueMemberS{Value: "active"},
	}

	walletItem := map[string]types.AttributeValue{
		"id":      &types.AttributeValueMemberS{Value: "wallet-abc"},
		"user_id": &types.AttributeValueMemberS{Value: "user-789"},
		"balance": &types.AttributeValueMemberS{Value: "500"},
		"version": &types.AttributeValueMemberN{Value: "1"},
	}

	db.On("GetItem", ctx, mock.Anything).Return(&dynamodb.GetItemOutput{Item: resItem}, nil).Once()
	db.On("Query", ctx, mock.Anything).Return(&dynamodb.QueryOutput{
		Items: []map[string]types.AttributeValue{walletItem},
	}, nil)
	db.On("UpdateItem", ctx, mock.Anything).Return(&dynamodb.UpdateItemOutput{}, nil).Twice()

	svc := New(db, nil, "wallets", "reservations", "")

	err := svc.ConfirmDeduction(ctx, "pay-456", "res-123", "gw-ref-xyz")

	assert.NoError(t, err)
	db.AssertExpectations(t)
}

func TestReleaseFunds_Success(t *testing.T) {
	ctx := context.Background()
	db := new(mockDB)

	resItem := map[string]types.AttributeValue{
		"id":     &types.AttributeValueMemberS{Value: "res-123"},
		"status": &types.AttributeValueMemberS{Value: "active"},
	}

	db.On("GetItem", ctx, mock.Anything).Return(&dynamodb.GetItemOutput{Item: resItem}, nil)
	db.On("UpdateItem", ctx, mock.Anything).Return(&dynamodb.UpdateItemOutput{}, nil)

	svc := New(db, nil, "wallets", "reservations", "")

	err := svc.ReleaseFunds(ctx, "res-123", "payment cancelled")

	assert.NoError(t, err)
	db.AssertExpectations(t)
}
