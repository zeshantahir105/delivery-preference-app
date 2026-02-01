package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/zeshan-weel/backend/internal/db"
	"github.com/zeshan-weel/backend/internal/handler"
	"github.com/zeshan-weel/backend/internal/middleware"
)

func main() {
	// Load .env from repo root (when run from backend/ via "go run ./cmd/server")
	_ = godotenv.Load("../.env")
	_ = godotenv.Load(".env")

	if err := db.RunMigrations(); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	pool, err := db.Open()
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	db.SeedTestUser(pool)

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret"
	}

	h := handler.New(pool, jwtSecret)
	auth := middleware.RequireAuth(jwtSecret)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/login", h.Login)
	mux.HandleFunc("GET /me", auth(h.Me))
	mux.HandleFunc("GET /orders", auth(h.ListOrders))
	mux.HandleFunc("POST /orders", auth(h.CreateOrder))
	mux.HandleFunc("GET /orders/{id}", auth(h.GetOrder))
	mux.HandleFunc("PUT /orders/{id}", auth(h.UpdateOrder))
	mux.HandleFunc("GET /orders/{id}/summary", auth(h.OrderSummary))

	// CORS for frontend
	cors := middleware.CORS(mux)

	addr := ":8080"
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, cors); err != nil {
		log.Fatalf("server: %v", err)
	}
}
