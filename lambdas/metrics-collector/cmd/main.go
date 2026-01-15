package main

import (
	"context"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"

	"github.com/HELL0ANTHONY/payment-system/lambdas/metrics-collector/internal/handler"
	"github.com/HELL0ANTHONY/payment-system/lambdas/metrics-collector/internal/service"
)

func main() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		panic(err)
	}

	cwClient := cloudwatch.NewFromConfig(cfg)

	namespace := os.Getenv("METRICS_NAMESPACE")
	if namespace == "" {
		namespace = "PaymentSystem"
	}

	svc := service.New(cwClient, namespace)
	h := handler.New(svc)

	lambda.Start(h.Handle)
}
