package errx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

func New(kind Kind, message string) *Error {
	return &Error{Kind: kind, Message: message}
}

func Newf(kind Kind, format string, args ...any) *Error {
	return &Error{Kind: kind, Message: fmt.Sprintf(format, args...)}
}

func resolveContextKind(cause error) (Kind, bool) {
	if errors.Is(cause, context.Canceled) {
		return KindCanceled, true
	}
	if errors.Is(cause, context.DeadlineExceeded) {
		return KindDeadlineExceeded, true
	}
	return 0, false
}

func Wrap(cause error, kind Kind, message string) *Error {
	if k, ok := resolveContextKind(cause); ok {
		return &Error{Kind: k, Message: message, Cause: cause}
	}
	return &Error{Kind: kind, Message: message, Cause: cause}
}

func Wrapf(cause error, kind Kind, format string, args ...any) *Error {
	if k, ok := resolveContextKind(cause); ok {
		return &Error{Kind: k, Message: fmt.Sprintf(format, args...), Cause: cause}
	}
	return &Error{Kind: kind, Message: fmt.Sprintf(format, args...), Cause: cause}
}

func IsKind(err error, kind Kind) bool {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind == kind
	}
	return false
}

func GetKind(err error) Kind {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind
	}
	return 0
}

func IsRetryable(err error) bool {
	if IsKind(err, KindCanceled) {
		return false
	}
	return IsKind(err, KindRateLimited) || IsKind(err, KindDeadlineExceeded) || IsKind(err, KindNetwork) || IsKind(err, KindInternal)
}

func FromHTTPStatus(status int) Kind {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return KindUnauthorized
	case status == http.StatusPaymentRequired:
		return KindUnauthorized
	case status == http.StatusTooManyRequests:
		return KindRateLimited
	case status >= 500:
		return KindInternal
	default:
		return KindInternal
	}
}
