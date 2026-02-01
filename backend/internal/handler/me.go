package handler

import (
	"encoding/json"
	"net/http"

	"github.com/zeshan-weel/backend/internal/middleware"
)

type MeResponse struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFrom(r.Context())
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var email string
	err := h.db.QueryRow("SELECT email FROM users WHERE id = $1", userID).Scan(&email)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MeResponse{ID: userID, Email: email})
}
