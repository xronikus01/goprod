package main

import (
	"context"
	"net/http"
	"strconv"
	"strings"
)

type ctxKey string

const userIDKey ctxKey = "user_id"

// AuthMiddleware оборачивает handler и требует валидный JWT в Authorization: Bearer <token>
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
		if tokenStr == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		claims, err := ValidateToken(tokenStr)
		if err != nil || claims == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// user id берём из Subject (sub)
		if claims.Subject == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		id, err := strconv.Atoi(claims.Subject)
		if err != nil || id <= 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, id)
		next(w, r.WithContext(ctx))
	}
}

// GetUserIDFromContext достаёт userID из контекста запроса
func GetUserIDFromContext(r *http.Request) (int, bool) {
	v := r.Context().Value(userIDKey)
	id, ok := v.(int)
	if !ok || id <= 0 {
		return 0, false
	}
	return id, true
}
