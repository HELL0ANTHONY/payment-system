package main

import (
	"context"
	"os"

	"github.com/HELL0ANTHONY/payment-system/shared/publisher"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"github.com/HELL0ANTHONY/payment-system/lambdas/gateway-processor/internal/handler"
	"github.com/HELL0ANTHONY/payment-system/lambdas/gateway-processor/internal/service"
)

func main() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		panic(err)
	}

	sqsClient := sqs.NewFromConfig(cfg)
	pub := publisher.NewSQS(sqsClient)

	gateway := service.NewMockGateway(0.1)

	svc := service.New(
		pub,
		gateway,
		os.Getenv("WALLET_QUEUE_URL"),
	)

	h := handler.New(svc)
	lambda.Start(h.Handle)
}
