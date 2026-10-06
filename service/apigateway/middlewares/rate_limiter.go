package middlewares

import (
	"net/http"
	"sync"

	"golang.org/x/time/rate"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/httpx"
)

type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rps      rate.Limit
	burst    int
}

func NewRateLimiter(rps int, burst int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rps:      rate.Limit(rps),
		burst:    burst,
	}
}

func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.RLock()
	limiter, exists := rl.limiters[ip]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		defer rl.mu.Unlock()

		// Double check in case another goroutine created it
		if limiter, exists = rl.limiters[ip]; exists {
			return limiter
		}

		limiter = rate.NewLimiter(rl.rps, rl.burst)
		rl.limiters[ip] = limiter
	}

	return limiter
}

func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := httpx.RealIP(r)
		limiter := rl.getLimiter(ip)

		if !limiter.Allow() {
			_ = httpx.JSON(w, http.StatusTooManyRequests, map[string]string{
				"error": "Too many requests from this IP, please try again later",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}
