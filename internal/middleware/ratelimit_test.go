package middleware

import (
	"net/http"
	"testing"
)

func TestDefaultKeyFuncUsesClientIPWithoutPort(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://example.com/api/users", nil)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}
	req.RemoteAddr = "127.0.0.1:54321"

	if got := DefaultKeyFunc(req); got != "127.0.0.1" {
		t.Fatalf("DefaultKeyFunc() = %q, want %q", got, "127.0.0.1")
	}
}

func TestDefaultKeyFuncPrefersForwardedHeader(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://example.com/api/users", nil)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}
	req.Header.Set("X-Forwarded-For", "203.0.113.15")
	req.RemoteAddr = "127.0.0.1:54321"

	if got := DefaultKeyFunc(req); got != "203.0.113.15" {
		t.Fatalf("DefaultKeyFunc() = %q, want %q", got, "203.0.113.15")
	}
}
