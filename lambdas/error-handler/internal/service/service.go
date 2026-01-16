package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/HELL0ANTHONY/payment-system/shared/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// DynamoDBClient defines the DynamoDB operations we need.
type DynamoDBClient interface {
	PutItem(
		ctx context.Context,
		params *dynamodb.PutItemInput,
		optFns ...func(*dynamodb.Options),
	) (*dynamodb.PutItemOutput, error)
}

// EventPublisher defines the event publishing operations we need.
type EventPublisher interface {
	Publish(ctx context.Context, queueURL string, event *events.Event) error
}

// FailedEvent represents a failed event stored for analysis.
type FailedEvent struct {
	CreatedAt     time.Time `dynamodbav:"created_at"`
	ID            string    `dynamodbav:"id"`
	OriginalEvent string    `dynamodbav:"original_event"`
	EventType     string    `dynamodbav:"event_type"`
	PaymentID     string    `dynamodbav:"payment_id"`
	ErrorMessage  string    `dynamodbav:"error_message"`
	Source        string    `dynamodbav:"source"`
	Status        string    `dynamodbav:"status"`
	RetryCount    int       `dynamodbav:"retry_count"`
}

type Service struct {
	db             DynamoDBClient
	publisher      EventPublisher
	tableName      string
	walletQueueURL string
	maxRetries     int
}

func New(
	db DynamoDBClient,
	pub EventPublisher,
	tableName, walletQueueURL string,
	maxRetries int,
) *Service {
	return &Service{
		db:             db,
		publisher:      pub,
		tableName:      tableName,
		walletQueueURL: walletQueueURL,
		maxRetries:     maxRetries,
	}
}

func (s *Service) HandleFailedEvent(
	ctx context.Context,
	messageID, body, source string,
	retryCount int,
) error {
	slog.Info(
		"handling failed event",
		"message_id",
		messageID,
		"source",
		source,
		"retry_count",
		retryCount,
	)

	var event events.Event
	if err := json.Unmarshal([]byte(body), &event); err != nil {
		slog.Error("failed to unmarshal event", "error", err)

		return s.storeFailedEvent(
			ctx,
			messageID,
			body,
			"unknown",
			"",
			"unmarshal error: "+err.Error(),
			source,
			retryCount,
		)
	}

	if retryCount < s.maxRetries && s.isRetryable(event.Type) {
		slog.Info("retrying event", "type", event.Type, "attempt", retryCount+1)
		return s.retryEvent(ctx, &event)
	}

	return s.storeFailedEvent(
		ctx,
		messageID,
		body,
		event.Type,
		event.PaymentID,
		"max retries exceeded",
		source,
		retryCount,
	)
}

func (s *Service) isRetryable(eventType string) bool {
	retryable := map[string]bool{
		events.PaymentInitiated: true,
		events.FundsReserved:    true,
	}

	return retryable[eventType]
}

func (s *Service) retryEvent(ctx context.Context, event *events.Event) error {
	if err := s.publisher.Publish(ctx, s.walletQueueURL, event); err != nil {
		slog.Error("failed to retry event", "error", err)
		return err
	}

	slog.Info("event retried", "type", event.Type, "payment_id", event.PaymentID)

	return nil
}

func (s *Service) storeFailedEvent(
	ctx context.Context,
	messageID, body, eventType, paymentID, errorMsg, source string,
	retryCount int,
) error {
	failed := &FailedEvent{
		ID:            messageID,
		OriginalEvent: body,
		EventType:     eventType,
		PaymentID:     paymentID,
		ErrorMessage:  errorMsg,
		Source:        source,
		RetryCount:    retryCount,
		Status:        "failed",
		CreatedAt:     time.Now().UTC(),
	}

	item, err := attributevalue.MarshalMap(failed)
	if err != nil {
		return fmt.Errorf("marshal failed event: %w", err)
	}

	_, err = s.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("store failed event: %w", err)
	}

	slog.Warn(
		"event stored as failed",
		"message_id",
		messageID,
		"event_type",
		eventType,
		"error",
		errorMsg,
	)

	return nil
}
