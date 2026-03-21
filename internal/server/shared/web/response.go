package web

import (
	"encoding/json"
	"net/http"
)

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func JSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func Error(w http.ResponseWriter, status int, message string) {
	ErrorWithCode(w, status, defaultErrorCode(status), message)
}

func ErrorWithCode(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, ErrorResponse{
		Code:    code,
		Message: message,
	})
}

func defaultErrorCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "bad_request"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusConflict:
		return "conflict"
	case http.StatusGone:
		return "gone"
	case http.StatusUnprocessableEntity:
		return "unprocessable_entity"
	case http.StatusServiceUnavailable:
		return "service_unavailable"
	case http.StatusBadGateway:
		return "bad_gateway"
	default:
		return "internal_error"
	}
}
