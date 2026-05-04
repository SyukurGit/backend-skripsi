package response

import "net/http"

type Envelope struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func OK(message string, data any) (int, Envelope) {
	return http.StatusOK, Envelope{Success: true, Message: message, Data: data}
}

func Created(message string, data any) (int, Envelope) {
	return http.StatusCreated, Envelope{Success: true, Message: message, Data: data}
}

func Error(code int, message string) (int, Envelope) {
	return code, Envelope{Success: false, Message: message}
}
