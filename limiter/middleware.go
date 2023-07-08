package limiter

import (
	"errors"
	"fmt"
	"net/http"
)

func RateLimiterMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		resp := IsRateLimited(r)

		if resp.Error != nil {
			if errors.Is(resp.Error, ErrNoMatchingRule) {
				next.ServeHTTP(w, r)
			}

			if errors.Is(resp.Error, ErrNoRateLimit) {
				next.ServeHTTP(w, r)
			}

			if errors.Is(resp.Error, ErrRateLimiterCheck) {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			// if rate limit exceeded
			if errors.Is(resp.Error, ErrTooManyRequests) {
				w.Header().Set("X-Ratelimit-Remaining", fmt.Sprintf("%d", resp.Header.XRatelimitRemaining))
				w.Header().Set("X-Ratelimit", fmt.Sprintf("%d", resp.Header.XRatelimit))
				w.Header().Set("X-Ratelimit-Retry-After", fmt.Sprintf("%d", resp.Header.XRateLimitRetryAfter))

				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
		}

		// if rate limit slot available
		w.Header().Set("X-Ratelimit-Remaining", fmt.Sprintf("%d", resp.Header.XRatelimitRemaining))
		w.Header().Set("X-Ratelimit", fmt.Sprintf("%d", resp.Header.XRatelimit))
		w.Header().Set("X-Ratelimit-Retry-After", fmt.Sprintf("%d", resp.Header.XRateLimitRetryAfter))

		next.ServeHTTP(w, r)
	})
}
