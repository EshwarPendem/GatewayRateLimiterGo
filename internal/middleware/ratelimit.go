package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/example/go-rate-limiter-gateway/internal/ratelimiter"
)

type RateLimitMiddleware struct {
	limiter ratelimiter.RateLimiter
	keyFunc func(r *http.Request) string
}

func NewRateLimitMiddleware(limiter ratelimiter.RateLimiter, keyFunc func(r *http.Request) string) *RateLimitMiddleware {
	if keyFunc == nil {
		keyFunc = DefaultKeyFunc
	}
	return &RateLimitMiddleware{
		limiter: limiter,
		keyFunc: keyFunc,
	}
}

func DefaultKeyFunc(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return strings.TrimSpace(realIP)
	}
	if r.RemoteAddr == "" {
		return "unknown"
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func (m *RateLimitMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := m.keyFunc(r)

		allowed, err := m.limiter.Allow(r.Context(), key)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if !allowed {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
