package controllers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"vibin_server/services"
)

type InteractionController struct {
	InteractionService *services.InteractionService
}

// NewInteractionController creates a new controller instance
func NewInteractionController(service *services.InteractionService) *InteractionController {
	return &InteractionController{InteractionService: service}
}

// HandleLikeUser processes a like request
func (c *InteractionController) HandleLikeUser(w http.ResponseWriter, r *http.Request) {
	var request struct {
		SenderHandle   string `json:"senderHandle"`
		ReceiverHandle string `json:"receiverHandle"`
	}

	// Parse request body
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	log.Printf("💖 %s liked %s", request.SenderHandle, request.ReceiverHandle)

	// Save like interaction
	err := c.InteractionService.SaveInteraction(context.TODO(), request.SenderHandle, request.ReceiverHandle, "like", "")
	if err != nil {
		http.Error(w, `{"error": "Failed to like user"}`, http.StatusInternalServerError)
		return
	}

	// Check if it's a match
	isMatch, err := c.InteractionService.IsMatch(context.TODO(), request.SenderHandle, request.ReceiverHandle)
	if err != nil {
		http.Error(w, `{"error": "Error checking match status"}`, http.StatusInternalServerError)
		return
	}

	// If matched, create a match record
	if isMatch {
		err = c.InteractionService.CreateMatch(context.TODO(), request.SenderHandle, request.ReceiverHandle)
		if err != nil {
			http.Error(w, `{"error": "Error creating match"}`, http.StatusInternalServerError)
			return
		}
	}

	// Send success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "User liked"})
}

// HandleDislikeUser processes a dislike request
func (c *InteractionController) HandleDislikeUser(w http.ResponseWriter, r *http.Request) {
	var request struct {
		SenderHandle   string `json:"senderHandle"`
		ReceiverHandle string `json:"receiverHandle"`
	}

	// Parse request body
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	log.Printf("💔 %s disliked %s", request.SenderHandle, request.ReceiverHandle)

	// Save dislike interaction
	err := c.InteractionService.SaveInteraction(context.TODO(), request.SenderHandle, request.ReceiverHandle, "dislike", "")
	if err != nil {
		http.Error(w, `{"error": "Failed to dislike user"}`, http.StatusInternalServerError)
		return
	}

	// Send success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "User disliked"})
}

// HandleGetInteractions fetches all interactions for a given receiverHandle
func (c *InteractionController) HandleGetInteractions(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ReceiverHandle string `json:"receiverHandle"`
	}

	// Decode request body
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Log request
	log.Printf("🔍 Fetching interactions for receiverHandle: %s", request.ReceiverHandle)

	// Fetch interactions
	interactions, err := c.InteractionService.GetInteractionsByReceiverHandle(context.TODO(), request.ReceiverHandle)
	if err != nil {
		http.Error(w, `{"error": "Failed to fetch interactions"}`, http.StatusInternalServerError)
		return
	}

	// Send JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(interactions)
}

// HandlePingUser processes a ping request
func (c *InteractionController) HandlePingUser(w http.ResponseWriter, r *http.Request) {
	var request struct {
		SenderHandle   string `json:"senderHandle"`
		ReceiverHandle string `json:"receiverHandle"`
		Message        string `json:"message,omitempty"` // ✅ Optional custom ping message
	}

	// Parse request body
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	log.Printf("📩 %s sent a ping to %s: %s", request.SenderHandle, request.ReceiverHandle, request.Message)

	// Save ping interaction
	err := c.InteractionService.SaveInteraction(
		context.TODO(),
		request.SenderHandle,
		request.ReceiverHandle,
		"ping",          // ✅ Use "ping" as the interaction type
		request.Message, // ✅ Save the custom message
	)
	if err != nil {
		http.Error(w, `{"error": "Failed to send ping"}`, http.StatusInternalServerError)
		return
	}

	// Send success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Ping sent successfully"})
}
