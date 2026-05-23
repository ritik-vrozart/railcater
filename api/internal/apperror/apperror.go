package apperror

import "errors"

var (
	ErrNotFound      = errors.New("not found")
	ErrConflict      = errors.New("conflict")
	ErrBadRequest    = errors.New("bad request")
	ErrInsufficient  = errors.New("insufficient stock")
)

type HTTPError struct {
	Status  int
	Message string
	Err     error
}

func (e *HTTPError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "error"
}

func (e *HTTPError) Unwrap() error { return e.Err }

func BadRequest(msg string) *HTTPError {
	return &HTTPError{Status: 400, Message: msg, Err: ErrBadRequest}
}

func NotFound(msg string) *HTTPError {
	return &HTTPError{Status: 404, Message: msg, Err: ErrNotFound}
}

func Conflict(msg string) *HTTPError {
	return &HTTPError{Status: 409, Message: msg, Err: ErrConflict}
}

func Unprocessable(msg string) *HTTPError {
	return &HTTPError{Status: 422, Message: msg, Err: ErrInsufficient}
}

func Unauthorized(msg string) *HTTPError {
	return &HTTPError{Status: 401, Message: msg}
}

func Internal(err error) *HTTPError {
	return &HTTPError{Status: 500, Message: "internal server error", Err: err}
}
