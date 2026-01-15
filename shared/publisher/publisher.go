package publisher

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"github.com/HELL0ANTHONY/payment-system/shared/events"
)

// SQS wraps the SQS client for publishing events.
type SQS struct {
	client *sqs.Client
}

// NewSQS creates a new SQS publisher.
func NewSQS(client *sqs.Client) *SQS {
	return &SQS{client: client}
}

// Publish sends an event to the specified queue.
func (p *SQS) Publish(ctx context.Context, queueURL string, event *events.Event) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = p.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(queueURL),
		MessageBody: aws.String(string(body)),
	})

	return err
}
