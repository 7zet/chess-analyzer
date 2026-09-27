package httpapi

import (
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

type visitor struct {
	count int
	reset time.Time
}
type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]visitor
	limit    int
	window   time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	r := &rateLimiter{visitors: make(map[string]visitor), limit: limit, window: window}
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for now := range ticker.C {
			r.mu.Lock()
			for key, value := range r.visitors {
				if now.After(value.reset) {
					delete(r.visitors, key)
				}
			}
			r.mu.Unlock()
		}
	}()
	return r
}

func (r *rateLimiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		host, _, err := net.SplitHostPort(req.RemoteAddr)
		if err != nil {
			host = req.RemoteAddr
		}
		now := time.Now()
		r.mu.Lock()
		v := r.visitors[host]
		if now.After(v.reset) {
			v = visitor{reset: now.Add(r.window)}
		}
		v.count++
		r.visitors[host] = v
		allowed := v.count <= r.limit
		remaining := max(0, r.limit-v.count)
		r.mu.Unlock()
		w.Header().Set("X-RateLimit-Limit", itoa(r.limit))
		w.Header().Set("X-RateLimit-Remaining", itoa(remaining))
		if !allowed {
			w.Header().Set("Retry-After", itoa(int(time.Until(v.reset).Seconds())))
			problem(w, http.StatusTooManyRequests, "Too many requests; please try again shortly")
			return
		}
		next.ServeHTTP(w, req)
	})
}

func (a *API) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-ID", middleware.GetReqID(r.Context()))
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		if a.cfg.Environment == "production" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		started := time.Now()
		next.ServeHTTP(ww, r)
		slog.Info("http request", "method", r.Method, "path", r.URL.Path, "status", ww.Status(), "bytes", ww.BytesWritten(), "duration", time.Since(started), "request_id", middleware.GetReqID(r.Context()))
	})
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	buffer := [20]byte{}
	index := len(buffer)
	for value > 0 {
		index--
		buffer[index] = byte('0' + value%10)
		value /= 10
	}
	return string(buffer[index:])
}
