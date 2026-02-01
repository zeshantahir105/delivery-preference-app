package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/zeshan-weel/backend/internal/middleware"
)

// aiHTTPTimeout is the timeout for OpenAI/Gemini API calls (generous for slow networks).
const aiHTTPTimeout = 45 * time.Second

// aiMaxOutputTokens allows full 2–3 sentence summaries (150 was truncating mid-sentence).
const aiMaxOutputTokens = 512

// fallbackSummaryText is shown when no AI worked (no keys set, or OpenAI/Gemini failed or returned empty).
const fallbackSummaryText = "Unable to generate Summary"

// OrderSummaryResponse is the JSON response for order summary (AI or fallback).
type OrderSummaryResponse struct {
	Summary string `json:"summary"`
	Source  string `json:"source,omitempty"` // "ai" or "fallback"
}

// OrderSummary returns an AI-generated or fallback summary of the order.
// Backend-proxied: uses OPENAI_API_KEY or GEMINI_API_KEY when set; otherwise returns a plain fallback.
// Disabled gracefully and mockable for tests (no key → fallback).
func (h *Handler) OrderSummary(w http.ResponseWriter, r *http.Request) {
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

	desc := orderDescription(id, preference, address, pickupTime, createdAt)
	summary, source := generateOrderSummary(desc)
	resp := OrderSummaryResponse{Summary: summary, Source: source}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// orderDescription builds a clear string with order number, preference, address, pickup time, creation date.
func orderDescription(id int, preference string, address sql.NullString, pickupTime sql.NullTime, createdAt time.Time) string {
	var b strings.Builder
	b.WriteString("Order number: ")
	b.WriteString(strconv.Itoa(id))
	b.WriteString(". Preference: ")
	b.WriteString(strings.ReplaceAll(preference, "_", " "))
	if address.Valid && address.String != "" {
		b.WriteString(". Address: ")
		b.WriteString(address.String)
	} else {
		b.WriteString(". Address: (none)")
	}
	if pickupTime.Valid {
		b.WriteString(". Pickup time: ")
		b.WriteString(pickupTime.Time.Format(time.RFC3339))
	} else {
		b.WriteString(". Pickup time: (none)")
	}
	b.WriteString(". Creation date: ")
	b.WriteString(createdAt.Format(time.RFC3339))
	return b.String()
}

func generateOrderSummary(orderDesc string) (summary, source string) {
	// Prompt: create the order summary and give order details (order number, preference, address, pickup time, creation date).
	prompt := "Create the order summary for the customer in one or two complete sentences. Include order number, preference, address, pickup time. Use the following order details: " + orderDesc

	// Try OpenAI first
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		log.Printf("order summary: input prompt: %s", prompt)
		s, err := callOpenAI(prompt, key)
		if err != nil {
			log.Printf("order summary: OpenAI call failed: %v", err)
			return fallbackSummaryText, "fallback"
		}
		if s == "" {
			log.Printf("order summary: OpenAI returned empty content, using fallback")
			return fallbackSummaryText, "fallback"
		}
		log.Printf("order summary: output (%d chars): %s", len(s), s)
		return s, "ai"
	}

	// Then Gemini
	if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		log.Printf("order summary: input prompt: %s", prompt)
		s, err := callGemini(prompt, key)
		if err != nil {
			log.Printf("order summary: Gemini call failed: %v", err)
			return fallbackSummaryText, "fallback"
		}
		if s == "" {
			log.Printf("order summary: Gemini returned empty content, using fallback")
			return fallbackSummaryText, "fallback"
		}
		log.Printf("order summary: output (%d chars): %s", len(s), s)
		return s, "ai"
	}

	// No AI key set; neither OpenAI nor Gemini used
	return fallbackSummaryText, "fallback"
}

// callOpenAI calls OpenAI Chat Completions and returns the first message content.
func callOpenAI(prompt, apiKey string) (string, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return "", errors.New("openai: empty API key")
	}
	reqBody := struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		MaxTokens int `json:"max_tokens,omitempty"`
	}{
		Model: "gpt-4o-mini",
		Messages: []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{Role: "user", Content: prompt},
		},
		MaxTokens: aiMaxOutputTokens,
	}
	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: aiHTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var errBody struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			} `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		msg := errBody.Error.Message
		if msg == "" {
			msg = resp.Status
		}
		return "", errors.New("openai " + strconv.Itoa(resp.StatusCode) + ": " + msg)
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", nil
	}
	// OpenAI returns a single content string per message (no parts array like Gemini); use first choice.
	return strings.TrimSpace(out.Choices[0].Message.Content), nil
}

// Gemini API: request/response structs and endpoint (net/http only; no external libs).
const geminiGenerateContentURL = "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent"

// GeminiGenerateContentRequest is the JSON body for generateContent.
type GeminiGenerateContentRequest struct {
	Contents         []GeminiContentItem   `json:"contents"`
	GenerationConfig *GeminiGenerationConfig `json:"generationConfig,omitempty"`
}

// GeminiContentItem represents one user message (one turn).
type GeminiContentItem struct {
	Parts []GeminiPart `json:"parts"`
}

// GeminiPart holds the prompt text.
type GeminiPart struct {
	Text string `json:"text"`
}

// GeminiGenerationConfig limits output length.
type GeminiGenerationConfig struct {
	MaxOutputTokens int `json:"maxOutputTokens,omitempty"`
}

// GeminiGenerateContentResponse is the JSON response from generateContent.
type GeminiGenerateContentResponse struct {
	Candidates []GeminiCandidate  `json:"candidates"`
	Error      *GeminiAPIError    `json:"error,omitempty"`
}

// GeminiCandidate holds one generated reply with content parts.
type GeminiCandidate struct {
	Content GeminiContent `json:"content"`
}

// GeminiContent holds the list of parts (e.g. one text part).
type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

// GeminiAPIError is returned when the API returns 4xx/5xx.
type GeminiAPIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// callGemini calls Gemini generateContent (gemini-1.5-flash). Reads API key from env only; uses net/http.
// Prompt format: "Make a summary of the order" + order details. Parses JSON response and returns AI text.
// Handles missing API key and HTTP/API errors.
func callGemini(prompt, apiKey string) (string, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return "", errors.New("gemini: missing GEMINI_API_KEY")
	}
	// Build request using request structs.
	reqBody := GeminiGenerateContentRequest{
		Contents: []GeminiContentItem{
			{Parts: []GeminiPart{{Text: prompt}}},
		},
		GenerationConfig: &GeminiGenerationConfig{MaxOutputTokens: aiMaxOutputTokens},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	// Key in query; do not hardcode.
	url := geminiGenerateContentURL + "?key=" + apiKey
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: aiHTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	// Parse JSON response using response structs.
	var out GeminiGenerateContentResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	// Handle HTTP error (missing key, invalid key, rate limit, etc.).
	if resp.StatusCode != http.StatusOK {
		msg := resp.Status
		if out.Error != nil && out.Error.Message != "" {
			msg = out.Error.Message
		}
		return "", errors.New("gemini " + strconv.Itoa(resp.StatusCode) + ": " + msg)
	}
	if len(out.Candidates) == 0 || len(out.Candidates[0].Content.Parts) == 0 {
		return "", nil
	}
	// Join all parts: Gemini may return multiple parts (e.g. "Here's your order" + full summary on next part).
	var full strings.Builder
	for _, p := range out.Candidates[0].Content.Parts {
		if p.Text != "" {
			full.WriteString(p.Text)
		}
	}
	return strings.TrimSpace(full.String()), nil
}
