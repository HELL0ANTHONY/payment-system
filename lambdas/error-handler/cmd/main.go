package main

import (
	"context"
	"os"
	"strconv"

	"github.com/HELL0ANTHONY/payment-system/shared/publisher"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"github.com/HELL0ANTHONY/payment-system/lambdas/error-handler/internal/handler"
	"github.com/HELL0ANTHONY/payment-system/lambdas/error-handler/internal/service"
)

func main() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		panic(err)
	}

	db := dynamodb.NewFromConfig(cfg)
	sqsClient := sqs.NewFromConfig(cfg)
	pub := publisher.NewSQS(sqsClient)

	maxRetries := 3
	if v := os.Getenv("MAX_RETRIES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			maxRetries = n
		}
	}

	svc := service.New(
		db,
		pub,
		os.Getenv("FAILED_EVENTS_TABLE"),
		os.Getenv("WALLET_QUEUE_URL"),
		maxRetries,
	)

	h := handler.New(svc)
	lambda.Start(h.Handle)
}
