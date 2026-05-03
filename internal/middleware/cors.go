package middleware

import (
	"net/http"
	"strings"
)

const corsVaryHeader = "Origin"

// CORS adds a minimal, explicit CORS policy for browser clients.
// Allowed origins are passed as a comma-separated list.
func CORS(allowedOrigins string) func(http.Handler) http.Handler {
	origins := parseAllowedOrigins(allowedOrigins)
	allowAny := len(origins) == 1 && origins[0] == "*"

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if allowAny {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				} else if isAllowedOrigin(origin, origins) {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Add("Vary", corsVaryHeader)
				}

				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func parseAllowedOrigins(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		origin := strings.TrimSpace(part)
		if origin == "" {
			continue
		}
		out = append(out, origin)
	}
	return out
}

func isAllowedOrigin(origin string, allowed []string) bool {
	for _, candidate := range allowed {
		if origin == candidate {
			return true
		}
	}
	return false
}
