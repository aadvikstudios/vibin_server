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
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if request.SenderHandle == "" || request.ReceiverHandle == "" || request.InteractionType == "" || request.Action == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Process interaction dynamically
	err := c.InteractionService.CreateOrUpdateInteraction(
		context.Background(),
		request.SenderHandle,
		request.ReceiverHandle,
		request.InteractionType,
		request.Action,
		request.Message, // Pass optional message if available
	)
	if err != nil {
		http.Error(w, "Failed to process interaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Send success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Interaction processed successfully"})
}

// GetUserInteractionsHandler fetches interactions for a specific user
func (c *InteractionController) GetUserInteractionsHandler(w http.ResponseWriter, r *http.Request) {
	userHandle := r.URL.Query().Get("userHandle")

	// Validate input
	if userHandle == "" {
		http.Error(w, "Missing userHandle parameter", http.StatusBadRequest)
		return
	}

	// Fetch interactions
	interactions, err := c.InteractionService.GetUserInteractions(context.Background(), userHandle)
	if err != nil {
		http.Error(w, "Failed to fetch interactions: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to JSON and send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(interactions)
}
