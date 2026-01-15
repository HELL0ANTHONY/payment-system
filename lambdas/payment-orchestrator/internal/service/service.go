package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/HELL0ANTHONY/payment-system/shared/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var ErrPaymentNotFound = errors.New("payment not found")

type DynamoDBClient interface {
	PutItem(
		ctx context.Context,
		params *dynamodb.PutItemInput,
		optFns ...func(*dynamodb.Options),
	) (*dynamodb.PutItemOutput, error)
	GetItem(
		ctx context.Context,
		params *dynamodb.GetItemInput,
		optFns ...func(*dynamodb.Options),
	) (*dynamodb.GetItemOutput, error)
}

type EventPublisher interface {
	Publish(ctx context.Context, queueURL string, event *events.Event) error
}

type Payment struct {
	CreatedAt   time.Time       `dynamodbav:"created_at"`
	UpdatedAt   time.Time       `dynamodbav:"updated_at"`
	ID          string          `dynamodbav:"id"`
	UserID      string          `dynamodbav:"user_id"`
	ServiceID   string          `dynamodbav:"service_id"`
	Amount      decimal.Decimal `dynamodbav:"amount"`
	Currency    string          `dynamodbav:"currency"`
	Status      string          `dynamodbav:"status"`
	Description string          `dynamodbav:"description"`
}

type Service struct {
	db             DynamoDBClient
	publisher      EventPublisher
	tableName      string
	walletQueueURL string
}

func New(db DynamoDBClient, pub EventPublisher, tableName, walletQueueURL string) *Service {
	return &Service{
		db:             db,
		publisher:      pub,
		tableName:      tableName,
		walletQueueURL: walletQueueURL,
	}
}

// CreatePayment creates a new payment record.
func (s *Service) CreatePayment(
	ctx context.Context,
	userID, serviceID, currency, description string,
	amount decimal.Decimal,
) (*Payment, error) {
	payment := &Payment{
		ID:          uuid.New().String(),
		UserID:      userID,
		ServiceID:   serviceID,
		Amount:      amount,
		Currency:    currency,
		Status:      "pending",
		Description: description,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	item, err := attributevalue.MarshalMap(payment)
	if err != nil {
		return nil, fmt.Errorf("marshal payment: %w", err)
	}

	_, err = s.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})
	if err != nil {
		return nil, fmt.Errorf("save payment: %w", err)
	}

	event := events.New(events.PaymentInitiated, payment.ID, payment.UserID)
	event.WithAmount(payment.Amount, payment.Currency)

	if err := s.publisher.Publish(ctx, s.walletQueueURL, &event); err != nil {
		slog.Error("failed to publish event", "error", err, "payment_id", payment.ID)
	}

	slog.Info("payment created", "payment_id", payment.ID, "user_id", userID)

	return payment, nil
}

// GetPayment retrieves a payment by its ID.
func (s *Service) GetPayment(ctx context.Context, id string) (*Payment, error) {
	result, err := s.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get payment: %w", err)
	}

	if result.Item == nil {
		return nil, ErrPaymentNotFound
	}

	var payment Payment
	if err := attributevalue.UnmarshalMap(result.Item, &payment); err != nil {
		return nil, fmt.Errorf("unmarshal payment: %w", err)
	}

	return &payment, nil
}
