package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type userIDKeyType string

const userIDKey userIDKeyType = "user_id"

var (
	// Per-user rate limiters.
	userLimiters = make(map[string]*rate.Limiter)
	limiterMutex sync.RWMutex
)

func getUserLimiter(userID string) *rate.Limiter {
	if strings.TrimSpace(userID) == "" {
		userID = "anonymous"
	}

	limiterMutex.RLock()
	limiter, exists := userLimiters[userID]
	limiterMutex.RUnlock()
	if exists {
		return limiter
	}

	limiterMutex.Lock()
	defer limiterMutex.Unlock()

	// Double-check after obtaining write lock.
	if limiter, exists = userLimiters[userID]; exists {
		return limiter
	}

	// 10 requests per minute per user with burst up to 10.
	limiter = rate.NewLimiter(rate.Every(6*time.Second), 10)
	userLimiters[userID] = limiter
	return limiter
}

func getUserIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if raw := ctx.Value(userIDKey); raw != nil {
		if userID, ok := raw.(string); ok {
			return strings.TrimSpace(userID)
		}
	}
	return ""
}

func RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := resolveUserID(r)
		if userID == "" {
			userID = "anonymous"
		}

		limiter := getUserLimiter(userID)
		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func resolveUserID(r *http.Request) string {
	if r == nil {
		return ""
	}

	if userID := getUserIDFromContext(r.Context()); userID != "" {
		return userID
	}

	if userID := strings.TrimSpace(r.Header.Get("X-User-ID")); userID != "" {
		return userID
	}

	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if token != "" {
			if userID, err := verifySupabaseJWT(token); err == nil && userID != "" {
				return userID
			}
		}
	}

	if userID := strings.TrimSpace(r.URL.Query().Get("user_id")); userID != "" {
		return userID
	}

	if userID := extractUserIDFromJSONBody(r); userID != "" {
		return userID
	}

	if ip := clientIP(r); ip != "" {
		return "ip:" + ip
	}

	return ""
}

func extractUserIDFromJSONBody(r *http.Request) string {
	if r == nil || r.Body == nil {
		return ""
	}

	switch r.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
	default:
		return ""
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return ""
	}
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	if len(bytes.TrimSpace(bodyBytes)) == 0 {
		return ""
	}

	var payload struct {
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		return ""
	}

	return strings.TrimSpace(payload.UserID)
}

func clientIP(r *http.Request) string {
	if r == nil {
		return ""
	}

	if fwd := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); fwd != "" {
		parts := strings.Split(fwd, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}
