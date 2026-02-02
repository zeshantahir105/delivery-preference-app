package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/joho/godotenv"
	"github.com/zeshan-weel/backend/internal/db"
	"github.com/zeshan-weel/backend/internal/middleware"
)

func init() {
	// Load .env from project root when running tests (e.g. "cd backend && go test")
	_ = godotenv.Load("../.env")
	_ = godotenv.Load(".env")
}

func testServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	pool, err := db.Open()
	if err != nil {
		t.Skipf("db not available: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	if err := db.RunMigrations(); err != nil {
		t.Skipf("migrations failed (db may not be available): %v", err)
	}
	
	// Seed test user for login
	db.SeedTestUser(pool)

	jwtSecret := "test-secret"
	h := New(pool, jwtSecret)
	auth := middleware.RequireAuth(jwtSecret)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/login", h.Login)
	mux.HandleFunc("GET /me", auth(h.Me))
	mux.HandleFunc("POST /orders", auth(h.CreateOrder))
	mux.HandleFunc("GET /orders/{id}", auth(h.GetOrder))
	mux.HandleFunc("PUT /orders/{id}", auth(h.UpdateOrder))
	mux.HandleFunc("GET /orders/{id}/summary", auth(h.OrderSummary))

	srv := httptest.NewServer(middleware.CORS(mux))
	t.Cleanup(srv.Close)

	// Login to get token
	loginBody := `{"email":"user@weel.com","password":"password"}`
	resp, err := http.Post(srv.URL+"/auth/login", "application/json", bytes.NewBufferString(loginBody))
	if err != nil {
		t.Fatalf("login request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login failed: %d", resp.StatusCode)
	}
	var loginResp struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	resp.Body.Close()
	return srv, loginResp.Token
}

func TestLoginSuccess(t *testing.T) {
	pool, err := db.Open()
	if err != nil {
		t.Skipf("db not available: %v", err)
	}
	defer pool.Close()
	if err := db.RunMigrations(); err != nil {
		t.Skipf("migrations failed (db may not be available): %v", err)
	}
	db.SeedTestUser(pool)

	h := New(pool, "test-secret")
	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/login", h.Login)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/auth/login", "application/json",
		bytes.NewBufferString(`{"email":"user@weel.com","password":"password"}`))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("want 200, got %d", resp.StatusCode)
	}
	var out struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Token == "" {
		t.Error("expected non-empty token")
	}
}

func TestLoginFailure(t *testing.T) {
	pool, err := db.Open()
	if err != nil {
		t.Skipf("db not available: %v", err)
	}
	defer pool.Close()
	if err := db.RunMigrations(); err != nil {
		t.Skipf("migrations failed (db may not be available): %v", err)
	}
	db.SeedTestUser(pool)

	h := New(pool, "test-secret")
	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/login", h.Login)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/auth/login", "application/json",
		bytes.NewBufferString(`{"email":"user@weel.com","password":"wrong"}`))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", resp.StatusCode)
	}
}

func TestAuthGuardBlocksUnauthenticated(t *testing.T) {
	srv, _ := testServer(t)

	resp, err := http.Get(srv.URL + "/me")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("want 401 without token, got %d", resp.StatusCode)
	}
}

func TestOrderValidationRejectsInvalidInput(t *testing.T) {
	srv, token := testServer(t)

	tests := []struct {
		name string
		body string
	}{
		{"past pickup_time", `{"preference":"DELIVERY","address":"123 Main","pickup_time":"2020-01-01T12:00:00Z"}`},
		{"missing address for DELIVERY", `{"preference":"DELIVERY","pickup_time":"2030-01-01T12:00:00Z"}`},
		{"invalid preference", `{"preference":"INVALID","address":"123"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodPost, srv.URL+"/orders", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("want 400, got %d", resp.StatusCode)
			}
		})
	}
}

func TestOrderSummaryRequiresAuth(t *testing.T) {
	srv, token := testServer(t)

	// Create an order so we have a valid order ID
	createBody := `{"preference":"IN_STORE"}`
	createReq, _ := http.NewRequest(http.MethodPost, srv.URL+"/orders", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)
	createResp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	defer createResp.Body.Close()
	if createResp.StatusCode != http.StatusCreated {
		t.Skipf("create order failed: %d", createResp.StatusCode)
	}
	var orderResp struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(createResp.Body).Decode(&orderResp); err != nil {
		t.Fatalf("decode order: %v", err)
	}
	orderID := orderResp.ID
	if orderID < 1 {
		t.Fatalf("expected order id >= 1, got %d", orderID)
	}

	// GET /orders/{id}/summary without token must return 401
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/orders/"+strconv.Itoa(orderID)+"/summary", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("summary request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("order summary without auth: want 401, got %d", resp.StatusCode)
	}
}

func TestOrderSummaryReturnsFallbackWhenNoAIKey(t *testing.T) {
	srv, token := testServer(t)

	// Create an order first
	createBody := `{"preference":"IN_STORE"}`
	createReq, _ := http.NewRequest(http.MethodPost, srv.URL+"/orders", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)
	createResp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	defer createResp.Body.Close()
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("create order want 201, got %d", createResp.StatusCode)
	}
	var orderResp struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(createResp.Body).Decode(&orderResp); err != nil {
		t.Fatalf("decode order: %v", err)
	}
	orderID := orderResp.ID
	if orderID < 1 {
		t.Fatalf("expected order id >= 1, got %d", orderID)
	}

	// Get summary (no AI key in test env → fallback)
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/orders/"+strconv.Itoa(orderID)+"/summary", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("summary request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("want 200, got %d", resp.StatusCode)
	}
	var summaryResp struct {
		Summary string `json:"summary"`
		Source  string `json:"source"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&summaryResp); err != nil {
		t.Fatalf("decode summary: %v", err)
	}
	if summaryResp.Summary == "" {
		t.Error("expected non-empty summary")
	}
	if summaryResp.Source != "fallback" {
		t.Errorf("expected source fallback when no AI key, got %q", summaryResp.Source)
	}
}
