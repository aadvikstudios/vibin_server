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

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	log.Printf("🔍 Fetching matches for user: %s", request.UserHandle)

	// Fetch matches from DynamoDB
	matches, err := c.MatchService.GetMatchesByUserHandle(context.TODO(), request.UserHandle)
	if err != nil {
		http.Error(w, `{"error": "Failed to fetch matches"}`, http.StatusInternalServerError)
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(matches)
}
