package apperr

import "net/http"

type Error struct {
	Status  int               `json:"-"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"errors,omitempty"`

	Cause error `json:"-"`
}

func (e *Error) Error() string {
	return e.Message
}

func Validation(fields map[string]string) *Error {
	return &Error{
		Status:  http.StatusUnprocessableEntity,
		Message: "validasi gagal",
		Fields:  fields,
	}
}

func BadRequest(message string) *Error {
	return &Error{Status: http.StatusBadRequest, Message: message}
}

func NotFound(message string) *Error {
	return &Error{Status: http.StatusNotFound, Message: message}
}

func Internal(cause error) *Error {
	return &Error{
		Status:  http.StatusInternalServerError,
		Message: "terjadi kesalahan pada server",
		Cause:   cause,
	}
}
