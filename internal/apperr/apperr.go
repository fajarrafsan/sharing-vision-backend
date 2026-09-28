package apperr

import "net/http"

// Error adalah error yang sudah tahu status HTTP dan bentuk response-nya.
// Error lain yang sampai ke handler dianggap 500.
type Error struct {
	Status  int               `json:"-"`
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`

	Cause error `json:"-"`
}

func (e *Error) Error() string {
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Cause
}

func Validation(fields map[string]string) *Error {
	return &Error{
		Status:  http.StatusUnprocessableEntity,
		Code:    "validation_failed",
		Message: "validasi gagal",
		Fields:  fields,
	}
}

func BadRequest(message string) *Error {
	return &Error{Status: http.StatusBadRequest, Code: "bad_request", Message: message}
}

func Unauthorized(message string) *Error {
	return &Error{Status: http.StatusUnauthorized, Code: "unauthorized", Message: message}
}

func Forbidden(message string) *Error {
	return &Error{Status: http.StatusForbidden, Code: "forbidden", Message: message}
}

func NotFound(message string) *Error {
	return &Error{Status: http.StatusNotFound, Code: "not_found", Message: message}
}

func MethodNotAllowed(message string) *Error {
	return &Error{Status: http.StatusMethodNotAllowed, Code: "method_not_allowed", Message: message}
}

func Conflict(message string) *Error {
	return &Error{Status: http.StatusConflict, Code: "conflict", Message: message}
}

func PayloadTooLarge(message string) *Error {
	return &Error{Status: http.StatusRequestEntityTooLarge, Code: "payload_too_large", Message: message}
}

func TooManyRequests(message string) *Error {
	return &Error{Status: http.StatusTooManyRequests, Code: "too_many_requests", Message: message}
}

func Internal(cause error) *Error {
	return &Error{
		Status:  http.StatusInternalServerError,
		Code:    "internal_error",
		Message: "terjadi kesalahan pada server",
		Cause:   cause,
	}
}
