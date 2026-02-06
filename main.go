package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using OS env")
	}

	log.Printf("DB_HOST=%q DB_PORT=%q DB_USER=%q DB_PASSWORD=%q DB_NAME=%q",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))

	// Инициализация JWT секретного ключа
	InitAuth()

	// TODO: Инициализация подключения к базе данных
	// Используйте функцию InitDB() из database.go
	if err := InitDB(); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer CloseDB()

	// TODO: Настройка HTTP маршрутов
	// Используйте обработчики из handlers.go
	http.HandleFunc("/register", RegisterHandler)
	http.HandleFunc("/login", LoginHandler)
	http.HandleFunc("/profile", AuthMiddleware(ProfileHandler))
	http.HandleFunc("/health", HealthHandler)

	// Запуск сервера
	port := getEnv("SERVER_PORT", "8080")
	log.Printf("🚀 Server starting on port %s", port)
	log.Printf("📝 Register: POST http://localhost:%s/register", port)
	log.Printf("🔐 Login: POST http://localhost:%s/login", port)
	log.Printf("👤 Profile: GET http://localhost:%s/profile (requires token)", port)
	log.Printf("❤️  Health: GET http://localhost:%s/health", port)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// getEnv получает значение переменной окружения или возвращает значение по умолчанию
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
