package main

import (

	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
	"github.com/mindgarden/server/internal/api"
	"github.com/mindgarden/server/internal/api/handlers"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Server startup failed: %v", err)
	}
}

func run() error {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment")
	}

	// Initialize database connection
	if err := handlers.InitDB(); err != nil {
		log.Printf("Warning: Database initialization failed: %v", err)
		log.Println("Server will continue but database-dependent endpoints will fail")
	}
	defer handlers.CloseDB()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"}, // Adjust for production
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Mount API routes
	log.Println("Mounting API routes...")
	api.MountRoutes(r)
	log.Println("API routes mounted via handlers.InitServices")

	log.Printf("Server starting on port %s", port)
	return http.ListenAndServe(":"+port, r)
}
