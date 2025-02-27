package controllers

import (
	"context"
	"encoding/json"
	"net/http"

	"vibin_server/services"
)

// InteractionController handles API requests related to interactions
type InteractionController struct {
	InteractionService *services.InteractionService
}

// CreateInteractionHandler processes interaction requests (like, ping, approval)
func (c *InteractionController) CreateInteractionHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		SenderHandle    string `json:"senderHandle"`
		ReceiverHandle  string `json:"receiverHandle"`
		InteractionType string `json:"interactionType"` // like, ping
		Action          string `json:"action"`          // like, dislike, approve, reject
	}

	// Decode request body
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if request.SenderHandle == "" || request.ReceiverHandle == "" || request.InteractionType == "" || request.Action == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Process interaction
	err := c.InteractionService.CreateOrUpdateInteraction(context.Background(), request.SenderHandle, request.ReceiverHandle, request.InteractionType, request.Action)
	if err != nil {
		http.Error(w, "Failed to process interaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Send success response
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Interaction processed successfully"}`))
}

// GetUserInteractionsHandler fetches interactions for a specific user
func (c *InteractionController) GetUserInteractionsHandler(w http.ResponseWriter, r *http.Request) {
	user := r.URL.Query().Get("userHandle")

	// Validate input
	if user == "" {
		http.Error(w, "Missing userHandle parameter", http.StatusBadRequest)
		return
	}

	// Fetch interactions
	interactions, err := c.InteractionService.GetUserInteractions(context.Background(), user)
	if err != nil {
		http.Error(w, "Failed to fetch interactions: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to JSON and send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(interactions)
}
