package httpserver

import (
	"encoding/json"
	"net/http"
)

type responseMeta struct {
	RequestID string `json:"requestId"`
}

type successResponse struct {
	Data any          `json:"data"`
	Meta responseMeta `json:"meta"`
}

type errorBody struct {
	Code        string            `json:"code"`
	Message     string            `json:"message"`
	FieldErrors map[string]string `json:"fieldErrors,omitempty"`
	RequestID   string            `json:"requestId"`
	Details     any               `json:"details,omitempty"`
}

type errorResponse struct {
	Error errorBody `json:"error"`
}

// RespondJSON writes a standardized success JSON envelope.
func RespondJSON(response http.ResponseWriter, request *http.Request, status int, data any) {
	writeJSON(response, status, successResponse{
		Data: data,
		Meta: responseMeta{RequestID: RequestIDFromContext(request.Context())},
	})
}

// RespondError writes a standardized error JSON envelope.
func RespondError(response http.ResponseWriter, request *http.Request, status int, code, message string, details any, fieldErrors map[string]string) {
	writeJSON(response, status, errorResponse{Error: errorBody{
		Code:        code,
		Message:     message,
		FieldErrors: fieldErrors,
		RequestID:   RequestIDFromContext(request.Context()),
		Details:     details,
	}})
}

func writeSuccess(response http.ResponseWriter, request *http.Request, status int, data any) {
	RespondJSON(response, request, status, data)
}

func writeError(response http.ResponseWriter, request *http.Request, status int, code, message string, details any) {
	RespondError(response, request, status, code, message, details, nil)
}

func writeJSON(response http.ResponseWriter, status int, body any) {
	response.Header().Set("Content-Type", "application/json")
	response.Header().Set("Cache-Control", "no-store")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(body)
}
