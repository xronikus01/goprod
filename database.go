package main

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/lib/pq"
)

// Глобальная переменная для подключения к БД
var db *sql.DB

// InitDB инициализирует подключение к базе данных
func InitDB() error {
	// TODO: Реализуйте подключение к PostgreSQL
	//
	// Что нужно сделать:
	// 1. Составьте строку подключения используя fmt.Sprintf()
	//    Формат: "host=%s port=%s user=%s password=%s dbname=%s sslmode=disable"
	// 2. Получите параметры из переменных окружения с помощью getEnv()
	// 3. Откройте соединение с sql.Open("postgres", connStr)
	// 4. Проверьте подключение с помощью db.Ping()
	// 5. Обработайте ошибки на каждом шаге
	//
	// Переменные окружения: DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_USER", "postgres"),
		getEnv("DB_PASSWORD", "postgres"),
		getEnv("DB_NAME", "secure_service"),
	)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	return nil
}

// CloseDB закрывает соединение с базой данных
func CloseDB() {
	if db != nil {
		db.Close()
	}
}

// CreateUser создает нового пользователя в базе данных
func CreateUser(email, username, passwordHash string) (*User, error) {
	const q = `
		INSERT INTO users (email, username, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`

	u := &User{
		Email:        email,
		Username:     username,
		PasswordHash: passwordHash,
	}

	if err := db.QueryRow(q, email, username, passwordHash).Scan(&u.ID, &u.CreatedAt); err != nil {
		return nil, err
	}

	return u, nil
}

// GetUserByEmail находит пользователя по email
func GetUserByEmail(email string) (*User, error) {
	const q = `
		SELECT id, email, username, password_hash, created_at
		FROM users
		WHERE email = $1
	`

	u := &User{}
	err := db.QueryRow(q, email).Scan(&u.ID, &u.Email, &u.Username, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return u, nil
}

// GetUserByID находит пользователя по ID
func GetUserByID(userID int) (*User, error) {
	const q = `
		SELECT id, email, username, created_at
		FROM users
		WHERE id = $1
	`

	u := &User{}
	err := db.QueryRow(q, userID).Scan(&u.ID, &u.Email, &u.Username, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return u, nil
}

// UserExistsByEmail проверяет, существует ли пользователь с данным email
func UserExistsByEmail(email string) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`
	var exists bool
	if err := db.QueryRow(q, email).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

// GetDB возвращает подключение к базе данных (для тестирования)
func GetDB() *sql.DB {
	return db
}
