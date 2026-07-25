package errx

import (
	"errors"
	"fmt"
)

type Kind uint8

const (
	KindInvalidArgument Kind = iota + 1
	KindUnauthorized
	KindRateLimited
	KindCanceled
	KindDeadlineExceeded
	KindInternal
	KindNetwork
)

func (k Kind) String() string {
	switch k {
	case KindInvalidArgument:
		return "InvalidArgument"
	case KindUnauthorized:
		return "Unauthorized"
	case KindRateLimited:
		return "RateLimited"
	case KindCanceled:
		return "Canceled"
	case KindDeadlineExceeded:
		return "DeadlineExceeded"
	case KindInternal:
		return "Internal"
	case KindNetwork:
		return "Network"
	default:
		return fmt.Sprintf("Unknown(%d)", k)
	}
}

type Error struct {
	Kind    Kind
	Message string
	Raw     any
	Cause   error
}

func (e *Error) Error() string {
	if e.Message == "" {
		return e.Kind.String()
	}
	return e.Kind.String() + ": " + e.Message
}

func (e *Error) Unwrap() error {
	return e.Cause
}

func (e *Error) Is(target error) bool {
	var t *Error
	return errors.As(target, &t) && t.Kind == e.Kind
}

var (
	ErrInvalidArgument   = &Error{Kind: KindInvalidArgument}
	ErrUnauthorized      = &Error{Kind: KindUnauthorized}
	ErrRateLimited       = &Error{Kind: KindRateLimited}
	ErrCanceled          = &Error{Kind: KindCanceled}
	ErrDeadlineExceeded  = &Error{Kind: KindDeadlineExceeded}
	ErrInternal          = &Error{Kind: KindInternal}
	ErrNetwork           = &Error{Kind: KindNetwork}
)
