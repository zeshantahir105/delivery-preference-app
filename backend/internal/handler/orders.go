package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zeshan-weel/backend/internal/middleware"
)

const (
	PrefInStore  = "IN_STORE"
	PrefDelivery = "DELIVERY"
	PrefCurbside = "CURBSIDE"
)

var validPrefs = map[string]bool{PrefInStore: true, PrefDelivery: true, PrefCurbside: true}

type OrderRequest struct {
	Preference  string  `json:"preference"`
	Address     *string `json:"address"`
	PickupTime  *string `json:"pickup_time"`
}

type OrderResponse struct {
	ID         int       `json:"id"`
	UserID     int       `json:"user_id"`
	Preference string    `json:"preference"`
	Address    *string   `json:"address,omitempty"`
	PickupTime *string   `json:"pickup_time,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFrom(r.Context())
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req OrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	if err := validateOrder(&req); err != nil {
		http.Error(w, `{"error":"`+escapeJSON(err.Error())+`"}`, http.StatusBadRequest)
		return
	}

	var address sql.NullString
	var pickupTime sql.NullTime
	if req.Address != nil {
		address = sql.NullString{String: *req.Address, Valid: true}
	}
	if req.PickupTime != nil {
		t, _ := time.Parse(time.RFC3339, *req.PickupTime)
		pickupTime = sql.NullTime{Time: t, Valid: true}
	}

	var id int
	var createdAt time.Time
	err := h.db.QueryRow(
		`INSERT INTO orders (user_id, preference, address, pickup_time) VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at`,
		userID, req.Preference, address, pickupTime,
	).Scan(&id, &createdAt)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	resp := orderToResponse(id, userID, req.Preference, req.Address, req.PickupTime, createdAt)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFrom(r.Context())
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	rows, err := h.db.Query(
		"SELECT id, preference, address, pickup_time, created_at FROM orders WHERE user_id = $1 ORDER BY created_at DESC",
		userID,
	)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var list []OrderResponse
	for rows.Next() {
		var id int
		var preference string
		var address sql.NullString
		var pickupTime sql.NullTime
		var createdAt time.Time
		if err := rows.Scan(&id, &preference, &address, &pickupTime, &createdAt); err != nil {
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}
		var addrPtr, timePtr *string
		if address.Valid {
			addrPtr = &address.String
		}
		if pickupTime.Valid {
			s := pickupTime.Time.Format(time.RFC3339)
			timePtr = &s
		}
		list = append(list, orderToResponse(id, userID, preference, addrPtr, timePtr, createdAt))
	}
	if err := rows.Err(); err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	if list == nil {
		list = []OrderResponse{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFrom(r.Context())
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 1 {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	var preference string
	var address sql.NullString
	var pickupTime sql.NullTime
	var createdAt time.Time
	err = h.db.QueryRow(
		"SELECT preference, address, pickup_time, created_at FROM orders WHERE id = $1 AND user_id = $2",
		id, userID,
	).Scan(&preference, &address, &pickupTime, &createdAt)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	var addrPtr *string
	var timePtr *string
	if address.Valid {
		addrPtr = &address.String
	}
	if pickupTime.Valid {
		s := pickupTime.Time.Format(time.RFC3339)
		timePtr = &s
	}
	resp := orderToResponse(id, userID, preference, addrPtr, timePtr, createdAt)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) UpdateOrder(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFrom(r.Context())
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 1 {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	var req OrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	if err := validateOrder(&req); err != nil {
		http.Error(w, `{"error":"`+escapeJSON(err.Error())+`"}`, http.StatusBadRequest)
		return
	}

	var address sql.NullString
	var pickupTime sql.NullTime
	if req.Address != nil {
		address = sql.NullString{String: *req.Address, Valid: true}
	}
	if req.PickupTime != nil {
		t, _ := time.Parse(time.RFC3339, *req.PickupTime)
		pickupTime = sql.NullTime{Time: t, Valid: true}
	}

	result, err := h.db.Exec(
		`UPDATE orders SET preference = $1, address = $2, pickup_time = $3 WHERE id = $4 AND user_id = $5`,
		req.Preference, address, pickupTime, id, userID,
	)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	var createdAt time.Time
	_ = h.db.QueryRow("SELECT created_at FROM orders WHERE id = $1", id).Scan(&createdAt)
	resp := orderToResponse(id, userID, req.Preference, req.Address, req.PickupTime, createdAt)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func validateOrder(req *OrderRequest) error {
	if !validPrefs[req.Preference] {
		return errValidation("preference must be IN_STORE, DELIVERY, or CURBSIDE")
	}
	switch req.Preference {
	case PrefDelivery, PrefCurbside:
		if req.Address == nil || strings.TrimSpace(*req.Address) == "" {
			return errValidation("address required for DELIVERY and CURBSIDE")
		}
	}
	if req.Preference != PrefInStore {
		if req.PickupTime == nil || *req.PickupTime == "" {
			return errValidation("pickup_time required when not IN_STORE")
		}
		t, err := time.Parse(time.RFC3339, *req.PickupTime)
		if err != nil {
			return errValidation("pickup_time must be RFC3339")
		}
		if !t.After(time.Now()) {
			return errValidation("pickup_time must be in the future")
		}
	}
	return nil
}

type errValidation string

func (e errValidation) Error() string { return string(e) }

func orderToResponse(id, userID int, pref string, addr, pt *string, createdAt time.Time) OrderResponse {
	resp := OrderResponse{ID: id, UserID: userID, Preference: pref, CreatedAt: createdAt}
	if addr != nil {
		resp.Address = addr
	}
	if pt != nil {
		resp.PickupTime = pt
	}
	return resp
}

func escapeJSON(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}
