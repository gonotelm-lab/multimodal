package errx

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestNew(t *testing.T) {
	e := New(KindInvalidArgument, "prompt is required")
	if e.Kind != KindInvalidArgument {
		t.Errorf("expected KindInvalidArgument, got %v", e.Kind)
	}
	if e.Message != "prompt is required" {
		t.Errorf("expected message 'prompt is required', got %q", e.Message)
	}
	expected := "invalid_argument: prompt is required"
	if e.Error() != expected {
		t.Errorf("expected %q, got %q", expected, e.Error())
	}
}

func TestWrap(t *testing.T) {
	cause := fmt.Errorf("underlying")
	e := Wrap(cause, KindNetwork, "call dashscope failed")
	if e.Kind != KindNetwork {
		t.Errorf("expected KindNetwork, got %v", e.Kind)
	}
	if !errors.Is(e, cause) {
		t.Error("errors.Is should find underlying cause")
	}
	if errors.Unwrap(e) != cause {
		t.Error("Unwrap should return cause")
	}
}

func TestIs(t *testing.T) {
	e := New(KindUnauthorized, "invalid api key")
	if !errors.Is(e, ErrUnauthorized) {
		t.Error("errors.Is should match sentinel error")
	}
	if errors.Is(e, ErrInvalidArgument) {
		t.Error("errors.Is should not match different kind")
	}
}

func TestChainedIs(t *testing.T) {
	cause := fmt.Errorf("dial tcp: connection refused")
	e := Wrap(cause, KindRateLimited, "too many requests")
	if !errors.Is(e, ErrRateLimited) {
		t.Error("errors.Is should match through chain")
	}
	if !errors.Is(e, cause) {
		t.Error("errors.Is should find cause through chain")
	}
}

func TestIsKind(t *testing.T) {
	e := New(KindInternal, "server error")
	if !IsKind(e, KindInternal) {
		t.Error("IsKind should return true")
	}
	if IsKind(e, KindNetwork) {
		t.Error("IsKind should return false for different kind")
	}
	if IsKind(nil, KindInternal) {
		t.Error("IsKind should return false for nil")
	}
	plain := fmt.Errorf("plain error")
	if IsKind(plain, KindInternal) {
		t.Error("IsKind should return false for plain error")
	}
}

func TestGetKind(t *testing.T) {
	e := New(KindRateLimited, "rate limited")
	if GetKind(e) != KindRateLimited {
		t.Errorf("expected KindRateLimited, got %v", GetKind(e))
	}
	if GetKind(nil) != 0 {
		t.Error("GetKind should return 0 for nil")
	}
	plain := fmt.Errorf("plain")
	if GetKind(plain) != 0 {
		t.Error("GetKind should return 0 for plain error")
	}
}

func TestIsRetryable(t *testing.T) {
	if !IsRetryable(New(KindRateLimited, "")) {
		t.Error("rate limited should be retryable")
	}
	if !IsRetryable(New(KindNetwork, "")) {
		t.Error("network should be retryable")
	}
	if !IsRetryable(New(KindInternal, "")) {
		t.Error("internal should be retryable")
	}
	if IsRetryable(New(KindUnauthorized, "")) {
		t.Error("unauthorized should not be retryable")
	}
	if IsRetryable(New(KindInvalidArgument, "")) {
		t.Error("invalid argument should not be retryable")
	}
}

func TestFromHTTPStatus(t *testing.T) {
	tests := []struct {
		status int
		kind   Kind
	}{
		{http.StatusUnauthorized, KindUnauthorized},
		{http.StatusForbidden, KindUnauthorized},
		{http.StatusPaymentRequired, KindUnauthorized},
		{http.StatusTooManyRequests, KindRateLimited},
		{http.StatusInternalServerError, KindInternal},
		{http.StatusBadGateway, KindInternal},
		{http.StatusServiceUnavailable, KindInternal},
		{http.StatusGatewayTimeout, KindInternal},
		{http.StatusBadRequest, KindInternal},
	}
	for _, tt := range tests {
		got := FromHTTPStatus(tt.status)
		if got != tt.kind {
			t.Errorf("FromHTTPStatus(%d) = %v, want %v", tt.status, got, tt.kind)
		}
	}
}

func TestDashScopeCodeToKind(t *testing.T) {
	tests := []struct {
		code string
		kind Kind
	}{
		{"Arrearage", KindUnauthorized},
		{"InvalidParameter", KindInvalidArgument},
		{"DataInspectionFailed", KindInvalidArgument},
		{"APIConnectionError", KindNetwork},
		{"AllocationQuota", KindRateLimited},
		{"Throttling", KindRateLimited},
		{"InvalidFile.DownloadFailed", KindInvalidArgument},
		{"InternalError", KindInternal},
		{"SomeUnknownCode", KindInternal},
	}
	for _, tt := range tests {
		got := DashScopeCodeToKind(tt.code)
		if got != tt.kind {
			t.Errorf("DashScopeCodeToKind(%q) = %v, want %v", tt.code, got, tt.kind)
		}
	}
}

func TestMiniMaxCodeToKind(t *testing.T) {
	tests := []struct {
		code int64
		kind Kind
	}{
		{1000, KindInternal},
		{1001, KindNetwork},
		{1002, KindRateLimited},
		{1004, KindUnauthorized},
		{1008, KindUnauthorized},
		{1024, KindInternal},
		{1026, KindInvalidArgument},
		{1027, KindInvalidArgument},
		{1033, KindInternal},
		{1039, KindInvalidArgument},
		{1041, KindRateLimited},
		{1042, KindInvalidArgument},
		{2013, KindInvalidArgument},
		{2045, KindRateLimited},
		{2049, KindUnauthorized},
		{9999, KindInternal},
	}
	for _, tt := range tests {
		got := MiniMaxCodeToKind(tt.code)
		if got != tt.kind {
			t.Errorf("MiniMaxCodeToKind(%d) = %v, want %v", tt.code, got, tt.kind)
		}
	}
}

func TestOpenAIErrorTypeToKind(t *testing.T) {
	tests := []struct {
		typ  string
		kind Kind
	}{
		{"invalid_request_error", KindInvalidArgument},
		{"authentication_error", KindUnauthorized},
		{"insufficient_quota", KindRateLimited},
		{"rate_limit_error", KindRateLimited},
		{"server_error", KindInternal},
		{"api_error", KindInternal},
		{"unknown_type", KindInternal},
	}
	for _, tt := range tests {
		got := OpenAIErrorTypeToKind(tt.typ)
		if got != tt.kind {
			t.Errorf("OpenAIErrorTypeToKind(%q) = %v, want %v", tt.typ, got, tt.kind)
		}
	}
}

func TestError_Unwrap(t *testing.T) {
	e := &Error{Kind: KindInternal, Message: "test"}
	if e.Unwrap() != nil {
		t.Error("Unwrap should return nil for no cause")
	}
	cause := fmt.Errorf("cause")
	e2 := &Error{Kind: KindInternal, Cause: cause}
	if e2.Unwrap() != cause {
		t.Error("Unwrap should return cause")
	}
}

func TestError_Error_formats(t *testing.T) {
	tests := []struct {
		e        *Error
		expected string
	}{
		{&Error{Kind: KindInvalidArgument}, "invalid_argument"},
		{New(KindUnauthorized, "bad api key"), "unauthorized: bad api key"},
		{New(KindNetwork, ""), "network"},
	}
	for _, tt := range tests {
		if tt.e.Error() != tt.expected {
			t.Errorf("Error() = %q, want %q", tt.e.Error(), tt.expected)
		}
	}
}
