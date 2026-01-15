package main

import (
	"context"
	"os"

	"github.com/HELL0ANTHONY/payment-system/shared/publisher"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"github.com/HELL0ANTHONY/payment-system/lambdas/wallet-service/internal/handler"
	"github.com/HELL0ANTHONY/payment-system/lambdas/wallet-service/internal/service"
)

func main() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		panic(err)
	}

	db := dynamodb.NewFromConfig(cfg)
	sqsClient := sqs.NewFromConfig(cfg)
	pub := publisher.NewSQS(sqsClient)

	svc := service.New(
		db,
		pub,
		os.Getenv("WALLETS_TABLE"),
		os.Getenv("RESERVATIONS_TABLE"),
		os.Getenv("GATEWAY_QUEUE_URL"),
	)

	h := handler.New(svc)
	lambda.Start(h.Handle)
}
