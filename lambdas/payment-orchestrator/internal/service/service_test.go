package service

import (
	"context"
	"errors"
	"testing"

	"github.com/HELL0ANTHONY/payment-system/shared/events"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mocks.

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

	return args.Get(0).(*dynamodb.UpdateItemOutput), args.Error(1)
}

type mockPublisher struct {
	mock.Mock
}

func (m *mockPublisher) Publish(ctx context.Context, queueURL string, event *events.Event) error {
	args := m.Called(ctx, queueURL, event)

	return args.Error(0)
}

// Tests.

func TestCreatePayment_Success(t *testing.T) {
	ctx := context.Background()
	db := new(mockDB)
	pub := new(mockPublisher)

	db.On("PutItem", ctx, mock.Anything).Return(&dynamodb.PutItemOutput{}, nil)
	pub.On("Publish", ctx, "http://queue", mock.Anything).Return(nil)

	svc := New(db, pub, "payments", "http://queue")

	payment, err := svc.CreatePayment(
		ctx,
		"user-123",
		"service-456",
		"USD",
		"Test",
		decimal.NewFromInt(100),
	)

	assert.NoError(t, err)
	assert.NotEmpty(t, payment.ID)
	assert.Equal(t, "user-123", payment.UserID)
	assert.Equal(t, "service-456", payment.ServiceID)
	assert.Equal(t, "USD", payment.Currency)
	assert.Equal(t, "pending", payment.Status)
	assert.True(t, payment.Amount.Equal(decimal.NewFromInt(100)))

	db.AssertExpectations(t)
	pub.AssertExpectations(t)
}

func TestCreatePayment_DBError(t *testing.T) {
	ctx := context.Background()
	db := new(mockDB)
	pub := new(mockPublisher)

	db.On("PutItem", ctx, mock.Anything).Return(nil, errors.New("db error"))

	svc := New(db, pub, "payments", "http://queue")

	payment, err := svc.CreatePayment(
		ctx,
		"user-123",
		"svc",
		"USD",
		"Test",
		decimal.NewFromInt(100),
	)

	assert.Error(t, err)
	assert.Nil(t, payment)
	assert.Contains(t, err.Error(), "save payment")
}

func TestCreatePayment_PublishError_StillSucceeds(t *testing.T) {
	ctx := context.Background()
	db := new(mockDB)
	pub := new(mockPublisher)

	db.On("PutItem", ctx, mock.Anything).Return(&dynamodb.PutItemOutput{}, nil)
	pub.On("Publish", ctx, "http://queue", mock.Anything).Return(errors.New("sqs error"))

	svc := New(db, pub, "payments", "http://queue")

	payment, err := svc.CreatePayment(
		ctx,
		"user-123",
		"svc",
		"USD",
		"Test",
		decimal.NewFromInt(100),
	)

	// Payment should still be created even if publish fails
	assert.NoError(t, err)
	assert.NotNil(t, payment)
}

func TestGetPayment_Success(t *testing.T) {
	ctx := context.Background()
	db := new(mockDB)

	existingPayment := &Payment{
		ID:       "pay-123",
		UserID:   "user-456",
		Amount:   decimal.NewFromInt(50),
		Currency: "MXN",
		Status:   "completed",
	}
	item, _ := attributevalue.MarshalMap(existingPayment)

	db.On("GetItem", ctx, mock.Anything).Return(&dynamodb.GetItemOutput{Item: item}, nil)

	svc := New(db, nil, "payments", "")

	payment, err := svc.GetPayment(ctx, "pay-123")

	assert.NoError(t, err)
	assert.Equal(t, "pay-123", payment.ID)
	assert.Equal(t, "user-456", payment.UserID)
	assert.Equal(t, "completed", payment.Status)
}

func TestGetPayment_NotFound(t *testing.T) {
	ctx := context.Background()
	db := new(mockDB)

	db.On("GetItem", ctx, mock.Anything).Return(&dynamodb.GetItemOutput{Item: nil}, nil)

	svc := New(db, nil, "payments", "")

	payment, err := svc.GetPayment(ctx, "non-existent")

	assert.ErrorIs(t, err, ErrPaymentNotFound)
	assert.Nil(t, payment)
}

func TestUpdateStatus_Success(t *testing.T) {
	ctx := context.Background()
	db := new(mockDB)

	db.On("UpdateItem", ctx, mock.Anything).Return(&dynamodb.UpdateItemOutput{}, nil)

	svc := New(db, nil, "payments", "")

	err := svc.UpdateStatus(ctx, "pay-123", "completed")

	assert.NoError(t, err)
	db.AssertExpectations(t)
}
