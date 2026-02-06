package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// requireJSON проверяет Content-Type запроса.
// По требованиям: ожидаем application/json, иначе 415.
func requireJSON(w http.ResponseWriter, r *http.Request) bool {
	ct := r.Header.Get("Content-Type")
	if !strings.HasPrefix(strings.ToLower(ct), "application/json") {
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// parseJSONStrict читает r.Body ОДИН раз, строго декодирует JSON и проверяет отсутствие "хвоста".
// ВАЖНО: чтобы не ломаться, если middleware уже частично прочитал Body, мы буферизуем его,
// восстанавливаем r.Body и декодируем из буфера.
func parseJSONStrict(r *http.Request, dst any) error {
	if r.Body == nil {
		return fmt.Errorf("request body is empty")
	}

	// Буферизуем всё тело. Это защищает от случаев, когда кто-то уже трогал r.Body.
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	// Восстанавливаем r.Body на будущее (на случай, если дальше кто-то захочет читать)
	r.Body = io.NopCloser(bytes.NewReader(b))

	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return err
	}

	// Второй decode должен дать EOF (иначе "хвост")
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("unexpected data after json")
		}
		return err
	}

	return nil
}

func validateRegisterRequest(req *RegisterRequest) error {
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Username = strings.TrimSpace(req.Username)

	if req.Email == "" {
		return fmt.Errorf("email is required")
	}
	if req.Username == "" {
		return fmt.Errorf("username is required")
	}
	if strings.TrimSpace(req.Password) == "" {
		return fmt.Errorf("password is required")
	}
	if len(req.Password) < 6 {
		return fmt.Errorf("password too short")
	}
	return nil
}

func validateLoginRequest(req *LoginRequest) error {
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	if req.Email == "" {
		return fmt.Errorf("email is required")
	}
	if strings.TrimSpace(req.Password) == "" {
		return fmt.Errorf("password is required")
	}
	return nil
}

// RegisterHandler POST /register
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !requireJSON(w, r) {
		return
	}

	var req RegisterRequest
	if err := parseJSONStrict(r, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := validateRegisterRequest(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	exists, err := UserExistsByEmail(req.Email)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if exists {
		http.Error(w, "user already exists", http.StatusConflict)
		return
	}

	hash, err := HashPassword(req.Password)
	if err != nil {
		http.Error(w, "failed to hash password", http.StatusInternalServerError)
		return
	}

	u, err := CreateUser(req.Email, req.Username, hash)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	token, err := GenerateToken(int64(u.ID), u.Username)
	if err != nil {
		http.Error(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"token": token,
		"user": map[string]any{
			"id":         u.ID,
			"email":      u.Email,
			"username":   u.Username,
			"created_at": u.CreatedAt,
		},
	})
}

// LoginHandler POST /login
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !requireJSON(w, r) {
		return
	}

	var req LoginRequest
	if err := parseJSONStrict(r, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := validateLoginRequest(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	u, err := GetUserByEmail(req.Email)
	if err != nil {
		// одинаковое сообщение на неверный email/пароль
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	if !CheckPassword(u.PasswordHash, req.Password) {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	token, err := GenerateToken(int64(u.ID), u.Username)
	if err != nil {
		http.Error(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token": token,
		"user": map[string]any{
			"id":         u.ID,
			"email":      u.Email,
			"username":   u.Username,
			"created_at": u.CreatedAt,
		},
	})
}

// ProfileHandler GET /profile (protected)
func ProfileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := GetUserIDFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	u, err := GetUserByID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":         u.ID,
		"email":      u.Email,
		"username":   u.Username,
		"created_at": u.CreatedAt,
	})
}

// HealthHandler GET /health
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	if db != nil {
		if err := db.Ping(); err != nil {
			http.Error(w, "Database connection failed", http.StatusServiceUnavailable)
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"message": "Service is running",
	})
}
