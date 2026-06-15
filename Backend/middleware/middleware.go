package middleware

import (
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

func SecurityHeadersMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "same-origin")
			w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
			w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
			w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data: blob: https:; media-src 'self' blob: https:; connect-src 'self' ws: wss:; style-src 'self' 'unsafe-inline'; script-src 'self'")
			next.ServeHTTP(w, r)
		})
	}
}

type loginAttempt struct {
	count       int
	windowStart time.Time
}

type responseStatusRecorder struct {
	http.ResponseWriter
	status int
}

func (recorder *responseStatusRecorder) WriteHeader(status int) {
	recorder.status = status
	recorder.ResponseWriter.WriteHeader(status)
}

func LoginRateLimit(maxAttempts int, window time.Duration) func(http.Handler) http.Handler {
	var mu sync.Mutex
	attempts := make(map[string]loginAttempt)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host := r.RemoteAddr
			if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]); forwarded != "" {
				host = forwarded
			} else if parsedHost, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
				host = parsedHost
			}

			now := time.Now()
			mu.Lock()
			attempt := attempts[host]
			if attempt.windowStart.IsZero() || now.Sub(attempt.windowStart) >= window {
				attempt = loginAttempt{windowStart: now}
			}
			if attempt.count >= maxAttempts {
				mu.Unlock()
				w.Header().Set("Retry-After", strconv.Itoa(int(time.Until(attempt.windowStart.Add(window)).Seconds())))
				http.Error(w, "Too many login attempts. Try again later.", http.StatusTooManyRequests)
				return
			}
			mu.Unlock()

			recorder := &responseStatusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(recorder, r)

			mu.Lock()
			defer mu.Unlock()
			if recorder.status >= http.StatusBadRequest {
				attempt.count++
				attempts[host] = attempt
			} else {
				delete(attempts, host)
			}
		})
	}
}

// CORSMiddleware creates CORS middleware with configuration
func CORSMiddleware(config struct {
	AllowedOrigins []string `json:"allowed_origins"`
	AllowedMethods []string `json:"allowed_methods"`
	AllowedHeaders []string `json:"allowed_headers"`
}) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Set CORS headers from configuration
			methods := strings.Join(config.AllowedMethods, ", ")
			headers := strings.Join(config.AllowedHeaders, ", ")

			// Determine allowed origin
			allowedOrigin := ""
			if origin != "" {
				for _, allowed := range config.AllowedOrigins {
					if allowed == "*" || allowed == origin {
						allowedOrigin = allowed
						break
					}
				}
			}

			if allowedOrigin != "" {
				w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
				w.Header().Set("Access-Control-Allow-Methods", methods)
				w.Header().Set("Access-Control-Allow-Headers", headers)
				w.Header().Add("Vary", "Origin")
			}

			w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				if origin != "" && allowedOrigin == "" {
					http.Error(w, "origin is not allowed", http.StatusForbidden)
					return
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// LoggingMiddleware creates middleware for logging requests
func LoggingMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			log.Printf("%s %s %s", r.Method, sanitizedRequestURI(r), time.Since(start))
		})
	}
}

func sanitizedRequestURI(r *http.Request) string {
	if r.URL == nil || r.URL.RawQuery == "" {
		return r.RequestURI
	}

	copyURL := *r.URL
	query, err := url.ParseQuery(copyURL.RawQuery)
	if err != nil {
		return copyURL.Path
	}
	for _, key := range []string{"token", "p", "t", "s", "apiKey"} {
		if query.Has(key) {
			query.Set(key, "[REDACTED]")
		}
	}
	copyURL.RawQuery = query.Encode()
	return copyURL.RequestURI()
}
