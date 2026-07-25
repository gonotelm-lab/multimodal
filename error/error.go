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
	KindInternal
	KindNetwork
)

func (k Kind) String() string {
	switch k {
	case KindInvalidArgument:
		return "invalid_argument"
	case KindUnauthorized:
		return "unauthorized"
	case KindRateLimited:
		return "rate_limited"
	case KindInternal:
		return "internal"
	case KindNetwork:
		return "network"
	default:
		return fmt.Sprintf("unknown(%d)", k)
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
	ErrInvalidArgument = &Error{Kind: KindInvalidArgument}
	ErrUnauthorized    = &Error{Kind: KindUnauthorized}
	ErrRateLimited     = &Error{Kind: KindRateLimited}
	ErrInternal        = &Error{Kind: KindInternal}
	ErrNetwork         = &Error{Kind: KindNetwork}
)
