package middleware

import (
	"net/http"

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
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.Header.Get("X-Real-IP")
	}
	if ip == "" {
		ip = r.RemoteAddr
	}
	return ip
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
