package controllers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"vibin_server/services"
)

// MatchController struct
type MatchController struct {
	MatchService *services.MatchService
}

// NewMatchController initializes the controller
func NewMatchController(service *services.MatchService) *MatchController {
	return &MatchController{MatchService: service}
}

// HandleGetMatches - Fetch all matches for a given userHandle
func (c *MatchController) HandleGetMatches(w http.ResponseWriter, r *http.Request) {
	var request struct {
		UserHandle string `json:"userHandle"`
	}

	// ✅ Validate & Decode request body
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Printf("❌ Invalid request body: %v", err)
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	// ✅ Validate user handle
	if request.UserHandle == "" {
		log.Println("❌ User handle is required")
		http.Error(w, `{"error": "userHandle is required"}`, http.StatusBadRequest)
		return
	}

	log.Printf("🔍 Fetching matches for user: %s", request.UserHandle)

	// ✅ Fetch matches with last message & unread status
	matches, err := c.MatchService.GetMatchesByUserHandle(context.TODO(), request.UserHandle)
	if err != nil {
		log.Printf("❌ Failed to fetch matches: %v", err)
		http.Error(w, `{"error": "Failed to fetch matches"}`, http.StatusInternalServerError)
		return
	}

	// ✅ Send response with last message & unread status
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(matches); err != nil {
		log.Printf("❌ Failed to encode response: %v", err)
		http.Error(w, `{"error": "Failed to encode response"}`, http.StatusInternalServerError)
	}
}
