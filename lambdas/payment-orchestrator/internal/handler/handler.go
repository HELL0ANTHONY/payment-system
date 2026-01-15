package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/aws/aws-lambda-go/events"

	"github.com/HELL0ANTHONY/payment-system/lambdas/payment-orchestrator/internal/service"
	"github.com/HELL0ANTHONY/payment-system/lambdas/payment-orchestrator/pkg/models"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Handle(
	ctx context.Context,
	req events.APIGatewayProxyRequest,
) (events.APIGatewayProxyResponse, error) {
	switch req.HTTPMethod {
	case http.MethodPost:
		return h.createPayment(ctx, req)
	case http.MethodGet:
		return h.getPayment(ctx, req)
	default:
		return h.response(http.StatusMethodNotAllowed, models.ErrorJSON("method not allowed")), nil
	}
}

func (h *Handler) createPayment(
	ctx context.Context,
	req events.APIGatewayProxyRequest,
) (events.APIGatewayProxyResponse, error) {
	var input models.CreatePaymentRequest
	if err := json.Unmarshal([]byte(req.Body), &input); err != nil {
		return h.response(http.StatusBadRequest, models.ErrorJSON("invalid json")), nil
	}

	if err := input.Validate(); err != nil {
		return h.response(http.StatusBadRequest, models.ErrorJSON(err.Error())), nil
	}

	payment, err := h.svc.CreatePayment(
		ctx,
		input.UserID,
		input.ServiceID,
		input.Currency,
		input.Description,
		input.Amount,
	)
	if err != nil {
		return h.response(
			http.StatusInternalServerError,
			models.ErrorJSON("failed to create payment"),
		), nil
	}

	dto := models.PaymentDTO{
		ID:          payment.ID,
		UserID:      payment.UserID,
		ServiceID:   payment.ServiceID,
		Amount:      payment.Amount.String(),
		Currency:    payment.Currency,
		Status:      payment.Status,
		Description: payment.Description,
		CreatedAt:   payment.CreatedAt,
	}

	return h.response(http.StatusAccepted, models.SuccessJSON(dto)), nil
}

func (h *Handler) getPayment(
	ctx context.Context,
	req events.APIGatewayProxyRequest,
) (events.APIGatewayProxyResponse, error) {
	id := req.PathParameters["id"]
	if id == "" {
		return h.response(http.StatusBadRequest, models.ErrorJSON("id is required")), nil
	}

	payment, err := h.svc.GetPayment(ctx, id)
	if err != nil {
		if errors.Is(err, service.ErrPaymentNotFound) {
			return h.response(http.StatusNotFound, models.ErrorJSON("payment not found")), nil
		}

		return h.response(
			http.StatusInternalServerError,
			models.ErrorJSON("failed to get payment"),
		), nil
	}

	dto := models.PaymentDTO{
		ID:          payment.ID,
		UserID:      payment.UserID,
		ServiceID:   payment.ServiceID,
		Amount:      payment.Amount.String(),
		Currency:    payment.Currency,
		Status:      payment.Status,
		Description: payment.Description,
		CreatedAt:   payment.CreatedAt,
	}

	return h.response(http.StatusOK, models.SuccessJSON(dto)), nil
}

func (h *Handler) response(status int, body string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Headers:    models.Headers(),
		Body:       body,
	}
}
