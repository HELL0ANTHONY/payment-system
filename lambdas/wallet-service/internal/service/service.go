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

var (
	ErrWalletNotFound    = errors.New("wallet not found")
	ErrInsufficientFunds = errors.New("insufficient funds")
)

// DynamoDBClient defines the DynamoDB operations we need.
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
	UpdateItem(
		ctx context.Context,
		params *dynamodb.UpdateItemInput,
		optFns ...func(*dynamodb.Options),
	) (*dynamodb.UpdateItemOutput, error)
	Query(
		ctx context.Context,
		params *dynamodb.QueryInput,
		optFns ...func(*dynamodb.Options),
	) (*dynamodb.QueryOutput, error)
}

// EventPublisher defines the event publishing operations we need.
type EventPublisher interface {
	Publish(ctx context.Context, queueURL string, event *events.Event) error
}

type Wallet struct {
	UpdatedAt time.Time `dynamodbav:"updated_at"`
	ID        string    `dynamodbav:"id"`
	UserID    string    `dynamodbav:"user_id"`
	Balance   string    `dynamodbav:"balance"`
	Currency  string    `dynamodbav:"currency"`
	Version   int       `dynamodbav:"version"`
}

type Reservation struct {
	ExpiresAt time.Time `dynamodbav:"expires_at"`
	CreatedAt time.Time `dynamodbav:"created_at"`
	ID        string    `dynamodbav:"id"`
	PaymentID string    `dynamodbav:"payment_id"`
	UserID    string    `dynamodbav:"user_id"`
	Amount    string    `dynamodbav:"amount"`
	Currency  string    `dynamodbav:"currency"`
	Status    string    `dynamodbav:"status"`
}

type Service struct {
	db                DynamoDBClient
	publisher         EventPublisher
	walletsTable      string
	reservationsTable string
	gatewayQueueURL   string
}

func New(
	db DynamoDBClient,
	pub EventPublisher,
	walletsTable, reservationsTable, gatewayQueueURL string,
) *Service {
	return &Service{
		db:                db,
		publisher:         pub,
		walletsTable:      walletsTable,
		reservationsTable: reservationsTable,
		gatewayQueueURL:   gatewayQueueURL,
	}
}

func (s *Service) ReserveFunds(
	ctx context.Context,
	paymentID, userID string,
	amount decimal.Decimal,
	currency string,
) error {
	wallet, err := s.getWalletByUser(ctx, userID)
	if err != nil {
		return s.publishReservationFailed(ctx, paymentID, userID, amount, currency, err.Error())
	}

	balance, _ := decimal.NewFromString(wallet.Balance)
	if balance.LessThan(amount) {
		return s.publishReservationFailed(
			ctx,
			paymentID,
			userID,
			amount,
			currency,
			"insufficient funds",
		)
	}

	reservation := &Reservation{
		ID:        uuid.New().String(),
		PaymentID: paymentID,
		UserID:    userID,
		Amount:    amount.String(),
		Currency:  currency,
		Status:    "active",
		ExpiresAt: time.Now().Add(15 * time.Minute),
		CreatedAt: time.Now().UTC(),
	}

	if err := s.saveReservation(ctx, reservation); err != nil {
		return fmt.Errorf("save reservation: %w", err)
	}

	event := events.New(events.FundsReserved, paymentID, userID)
	event.WithAmount(amount, currency).WithReservation(reservation.ID)

	if err := s.publisher.Publish(ctx, s.gatewayQueueURL, &event); err != nil {
		slog.Error("failed to publish funds reserved", "error", err)

		return err
	}

	slog.Info("funds reserved", "payment_id", paymentID, "reservation_id", reservation.ID)

	return nil
}

func (s *Service) ConfirmDeduction(
	ctx context.Context,
	paymentID, reservationID, gatewayRef string,
) error {
	reservation, err := s.getReservation(ctx, reservationID)
	if err != nil {
		return err
	}

	amount, _ := decimal.NewFromString(reservation.Amount)
	if err := s.deductFromWallet(ctx, reservation.UserID, amount); err != nil {
		return err
	}

	reservation.Status = "confirmed"
	if err := s.updateReservation(ctx, reservation); err != nil {
		return err
	}

	slog.Info("funds deducted", "payment_id", paymentID, "amount", reservation.Amount)

	return nil
}

func (s *Service) ReleaseFunds(ctx context.Context, reservationID, reason string) error {
	reservation, err := s.getReservation(ctx, reservationID)
	if err != nil {
		return err
	}

	reservation.Status = "released"
	if err := s.updateReservation(ctx, reservation); err != nil {
		return err
	}

	slog.Info("funds released", "reservation_id", reservationID, "reason", reason)

	return nil
}

func (s *Service) getWalletByUser(ctx context.Context, userID string) (*Wallet, error) {
	result, err := s.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.walletsTable),
		IndexName:              aws.String("user_id-index"),
		KeyConditionExpression: aws.String("user_id = :uid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid": &types.AttributeValueMemberS{Value: userID},
		},
		Limit: aws.Int32(1),
	})
	if err != nil {
		return nil, err
	}

	if len(result.Items) == 0 {
		return nil, ErrWalletNotFound
	}

	var wallet Wallet
	if err := attributevalue.UnmarshalMap(result.Items[0], &wallet); err != nil {
		return nil, err
	}

	return &wallet, nil
}

func (s *Service) deductFromWallet(
	ctx context.Context,
	userID string,
	amount decimal.Decimal,
) error {
	wallet, err := s.getWalletByUser(ctx, userID)
	if err != nil {
		return err
	}

	_, err = s.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.walletsTable),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: wallet.ID},
		},
		UpdateExpression: aws.String(
			"SET balance = balance - :amount, updated_at = :now, version = version + :one",
		),
		ConditionExpression: aws.String("version = :v AND balance >= :amount"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":amount": &types.AttributeValueMemberN{Value: amount.String()},
			":now":    &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
			":one":    &types.AttributeValueMemberN{Value: "1"},
			":v":      &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", wallet.Version)},
		},
	})

	return err
}

func (s *Service) saveReservation(ctx context.Context, r *Reservation) error {
	item, err := attributevalue.MarshalMap(r)
	if err != nil {
		return err
	}

	_, err = s.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.reservationsTable),
		Item:      item,
	})

	return err
}

func (s *Service) getReservation(ctx context.Context, id string) (*Reservation, error) {
	result, err := s.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.reservationsTable),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, errors.New("reservation not found")
	}

	var r Reservation

	return &r, attributevalue.UnmarshalMap(result.Item, &r)
}

func (s *Service) updateReservation(ctx context.Context, r *Reservation) error {
	_, err := s.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.reservationsTable),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: r.ID},
		},
		UpdateExpression: aws.String("SET #status = :status"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status": &types.AttributeValueMemberS{Value: r.Status},
		},
	})

	return err
}

func (s *Service) publishReservationFailed(
	_ context.Context,
	paymentID, userID string,
	amount decimal.Decimal,
	currency, reason string,
) error {
	event := events.New(events.FundsReservationFailed, paymentID, userID)
	event.WithAmount(amount, currency).WithReason(reason)

	slog.Warn("reservation failed", "payment_id", paymentID, "reason", reason)

	return fmt.Errorf("reservation failed: %s", reason)
}
