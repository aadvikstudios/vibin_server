package controllers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"vibin_server/services"
)

// InteractionController handles API requests related to interactions
type InteractionController struct {
	InteractionService *services.InteractionService
}

// CreateInteractionHandler processes interaction requests (like, ping, approval, etc.)
func (c *InteractionController) CreateInteractionHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		SenderHandle    string  `json:"senderHandle"`
		ReceiverHandle  string  `json:"receiverHandle"`
		InteractionType string  `json:"interactionType"` // like, ping, invite
		Action          string  `json:"action"`          // like, dislike, approve, reject
		Message         *string `json:"message,omitempty"`
	}

	// Decode request body
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Println("❌ Invalid request payload:", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if request.SenderHandle == "" || request.ReceiverHandle == "" || request.InteractionType == "" || request.Action == "" {
		log.Println("⚠️ Missing required fields in request")
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Set a timeout for database operations
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Process interaction dynamically
	err := c.InteractionService.CreateOrUpdateInteraction(
		ctx,
		request.SenderHandle,
		request.ReceiverHandle,
		request.InteractionType,
		request.Action,
		request.Message, // Pass optional message if available
	)
	if err != nil {
		log.Printf("❌ Failed to process interaction: %v", err)
		http.Error(w, "Failed to process interaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Send success response
	response := map[string]string{"message": "Interaction processed successfully"}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetUserInteractionsHandler fetches all interactions for a specific user
func (c *InteractionController) GetUserInteractionsHandler(w http.ResponseWriter, r *http.Request) {
	userHandle := r.URL.Query().Get("userHandle")

	// Validate input
	if userHandle == "" {
		http.Error(w, "Missing userHandle parameter", http.StatusBadRequest)
		return
	}

	// Set a timeout for database operations
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Fetch interactions
	interactions, err := c.InteractionService.GetUserInteractions(ctx, userHandle)
	if err != nil {
		log.Printf("❌ Failed to fetch interactions for %s: %v", userHandle, err)
		http.Error(w, "Failed to fetch interactions: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to JSON and send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(interactions)
}

// GetMutualMatchesHandler fetches all mutual matches for a user
func (c *InteractionController) GetMutualMatchesHandler(w http.ResponseWriter, r *http.Request) {
	userHandle := r.URL.Query().Get("userHandle")

	// Validate input
	if userHandle == "" {
		http.Error(w, "Missing userHandle parameter", http.StatusBadRequest)
		return
	}

	// Set a timeout for database operations
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Fetch mutual matches
	matches, err := c.InteractionService.GetMutualMatches(ctx, userHandle)
	if err != nil {
		log.Printf("❌ Failed to fetch mutual matches for %s: %v", userHandle, err)
		http.Error(w, "Failed to fetch mutual matches: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to JSON and send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(matches)
}
