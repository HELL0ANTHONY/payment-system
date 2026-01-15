package models

import "encoding/json"

// Response represents the API response structure.
type Response struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

// ResponseHeaders returns the standard CORS headers for API responses.
func ResponseHeaders() map[string]string {
	return map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Headers": "Access-Control-Allow-Origin, Access-Control-Allow-Methods, Content-Type",
		"Content-Type":                 "application/json",
		"Access-Control-Allow-Methods": "POST",
	}
}

// SuccessResponse returns a JSON string for successful operations.
func SuccessResponse(message string) string {
	res, _ := json.Marshal(Response{
		Message: message,
		Status:  "OK",
	})
	return string(res)
}

// ErrorResponse returns a JSON string for error responses.
func ErrorResponse(errorMsg string) string {
	res, _ := json.Marshal(Response{
		Message: errorMsg,
		Status:  "ERROR_VALIDACION",
	})
	return string(res)
}
