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

func writeSuccess(response http.ResponseWriter, request *http.Request, status int, data any) {
	writeJSON(response, status, successResponse{
		Data: data,
		Meta: responseMeta{RequestID: RequestIDFromContext(request.Context())},
	})
}

func writeError(response http.ResponseWriter, request *http.Request, status int, code, message string, details any) {
	writeJSON(response, status, errorResponse{Error: errorBody{
		Code:      code,
		Message:   message,
		RequestID: RequestIDFromContext(request.Context()),
		Details:   details,
	}})
}

func writeJSON(response http.ResponseWriter, status int, body any) {
	response.Header().Set("Content-Type", "application/json")
	response.Header().Set("Cache-Control", "no-store")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(body)
}
