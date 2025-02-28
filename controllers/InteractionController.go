package controllers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"vibin_server/models"
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
	log.Printf("🔍 Received interaction request: Sender=%s, Receiver=%s, Type=%s, Action=%s",
		request.SenderHandle, request.ReceiverHandle, request.InteractionType, request.Action)

	// Set a timeout for database operations
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Process interaction dynamically
	isMatch, matchedProfile, err := c.InteractionService.CreateOrUpdateInteraction(
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
	response := map[string]interface{}{
		"message":        "Interaction processed successfully",
		"isMatch":        isMatch,
		"matchedProfile": matchedProfile,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
func (c *InteractionController) ApprovePingHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		SenderHandle   string `json:"senderHandle"`
		ReceiverHandle string `json:"receiverHandle"`
	}

	// Decode request body
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Println("❌ Invalid approve ping request:", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	log.Printf("✅ Approving ping from %s -> %s", request.SenderHandle, request.ReceiverHandle)

	// Process approval in InteractionService
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := c.InteractionService.HandlePingApproval(ctx, request.SenderHandle, request.ReceiverHandle)
	if err != nil {
		log.Printf("❌ Failed to approve ping: %v", err)
		http.Error(w, "Failed to approve ping: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Send success response
	response := map[string]string{"message": "Ping approved successfully"}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
func (c *InteractionController) DeclinePingHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		SenderHandle   string `json:"senderHandle"`
		ReceiverHandle string `json:"receiverHandle"`
	}

	// Decode request body
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Println("❌ Invalid decline ping request:", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	log.Printf("🚫 Declining ping from %s -> %s", request.SenderHandle, request.ReceiverHandle)

	// Process decline in InteractionService
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := c.InteractionService.HandlePingDecline(ctx, request.SenderHandle, request.ReceiverHandle)
	if err != nil {
		log.Printf("❌ Failed to decline ping: %v", err)
		http.Error(w, "Failed to decline ping: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Send success response
	response := map[string]string{"message": "Ping declined successfully"}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
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

	// Fetch mutual matches (with minimal profile data)
	matches, err := c.InteractionService.GetMutualMatches(ctx, userHandle)
	if err != nil {
		log.Printf("❌ Failed to fetch mutual matches for %s: %v", userHandle, err)
		http.Error(w, "Failed to fetch mutual matches: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to JSON and send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if matches == nil {
		matches = []models.MatchedUserDetails{}
	}
	json.NewEncoder(w).Encode(struct {
		Matches []models.MatchedUserDetails `json:"matches"`
	}{matches})

}

// GetSentInteractionsHandler fetches all sent interactions for a user
func (c *InteractionController) GetSentInteractionsHandler(w http.ResponseWriter, r *http.Request) {
	userHandle := r.URL.Query().Get("userHandle")

	// Validate input
	if userHandle == "" {
		http.Error(w, "Missing userHandle parameter", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Fetch sent interactions with user profile data
	interactions, err := c.InteractionService.GetUserInteractions(ctx, userHandle)
	if err != nil {
		log.Printf("❌ Failed to fetch sent interactions for %s: %v", userHandle, err)
		http.Error(w, "Failed to fetch interactions: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to JSON and send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(struct {
		Interactions []models.InteractionWithProfile `json:"interactions"`
	}{interactions})
}

// GetReceivedInteractionsHandler fetches all received interactions for a user
func (c *InteractionController) GetReceivedInteractionsHandler(w http.ResponseWriter, r *http.Request) {
	userHandle := r.URL.Query().Get("userHandle")

	// Validate input
	if userHandle == "" {
		http.Error(w, "Missing userHandle parameter", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Fetch received interactions with user profile data
	interactions, err := c.InteractionService.GetReceivedInteractions(ctx, userHandle)
	if err != nil {
		log.Printf("❌ Failed to fetch received interactions for %s: %v", userHandle, err)
		http.Error(w, "Failed to fetch interactions: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to JSON and send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(struct {
		Interactions []models.InteractionWithProfile `json:"interactions"`
	}{interactions})
}
