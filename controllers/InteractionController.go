package controllers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"vibin_server/services"
)

// InteractionController struct
type InteractionController struct {
	InteractionService *services.InteractionService
}

// NewInteractionController initializes the controller
func NewInteractionController(service *services.InteractionService) *InteractionController {
	return &InteractionController{InteractionService: service}
}

// HandleLikeUser - User likes another user
func (c *InteractionController) HandleLikeUser(w http.ResponseWriter, r *http.Request) {
	var request struct {
		SenderHandle   string `json:"senderHandle"`
		ReceiverHandle string `json:"receiverHandle"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	log.Printf("💖 %s liked %s", request.SenderHandle, request.ReceiverHandle)

	err := c.InteractionService.SaveInteraction(context.TODO(), request.SenderHandle, request.ReceiverHandle, "like", "")
	if err != nil {
		http.Error(w, `{"error": "Failed to like user"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "User liked"})
}

// HandleDislikeUser - User dislikes another user
func (c *InteractionController) HandleDislikeUser(w http.ResponseWriter, r *http.Request) {
	var request struct {
		SenderHandle   string `json:"senderHandle"`
		ReceiverHandle string `json:"receiverHandle"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	log.Printf("💔 %s disliked %s", request.SenderHandle, request.ReceiverHandle)

	err := c.InteractionService.SaveInteraction(context.TODO(), request.SenderHandle, request.ReceiverHandle, "dislike", "")
	if err != nil {
		http.Error(w, `{"error": "Failed to dislike user"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "User disliked"})
}

// HandlePingUser - User sends a ping
func (c *InteractionController) HandlePingUser(w http.ResponseWriter, r *http.Request) {
	var request struct {
		SenderHandle   string `json:"senderHandle"`
		ReceiverHandle string `json:"receiverHandle"`
		Message        string `json:"message,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	log.Printf("📩 %s sent a ping to %s: %s", request.SenderHandle, request.ReceiverHandle, request.Message)

	err := c.InteractionService.SaveInteraction(context.TODO(), request.SenderHandle, request.ReceiverHandle, "ping", request.Message)
	if err != nil {
		http.Error(w, `{"error": "Failed to send ping"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Ping sent successfully"})
}

// HandleApprovePing - User approves a ping request
func (c *InteractionController) HandleApprovePing(w http.ResponseWriter, r *http.Request) {
	var request struct {
		SenderHandle   string `json:"senderHandle"`
		ReceiverHandle string `json:"receiverHandle"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	log.Printf("✅ %s approved ping from %s", request.ReceiverHandle, request.SenderHandle)

	ctx := context.TODO()

	// ✅ Step 1: Fetch the original ping message from the interaction table
	originalMessage, err := c.InteractionService.GetPingMessage(ctx, request.SenderHandle, request.ReceiverHandle)
	if err != nil {
		log.Printf("⚠️ Failed to fetch original ping message: %v", err)
		originalMessage = "" // Use an empty message if fetching fails
	}

	// ✅ Step 2: Update interaction status to "approved"
	if err := c.InteractionService.UpdateInteractionStatus(ctx, request.SenderHandle, request.ReceiverHandle, "approved", "ping"); err != nil {
		log.Printf("❌ Failed to update interaction status: %v", err)
		http.Error(w, `{"error": "Failed to approve ping"}`, http.StatusInternalServerError)
		return
	}

	// ✅ Step 3: Create a match and insert the original ping message
	if err := c.InteractionService.HandleMatch(ctx, []string{request.SenderHandle, request.ReceiverHandle}, originalMessage); err != nil {
		log.Printf("❌ Failed to handle match creation: %v", err)
		http.Error(w, `{"error": "Failed to create match"}`, http.StatusInternalServerError)
		return
	}

	// ✅ Step 4: Success Response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Ping approved, match created, and initial message sent",
	})
}

// HandleDeclinePing - User declines a ping request
func (c *InteractionController) HandleDeclinePing(w http.ResponseWriter, r *http.Request) {
	var request struct {
		SenderHandle   string `json:"senderHandle"`
		ReceiverHandle string `json:"receiverHandle"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	log.Printf("❌ %s declined ping from %s", request.ReceiverHandle, request.SenderHandle)

	// ✅ Update interaction status from "pending" to "declined"
	err := c.InteractionService.UpdateInteractionStatus(context.TODO(), request.SenderHandle, request.ReceiverHandle, "declined", "ping")
	if err != nil {
		http.Error(w, `{"error": "Failed to decline ping"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Ping declined"})
}

// HandleGetInteractions - Fetch all interactions for a user
func (c *InteractionController) HandleGetInteractions(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ReceiverHandle string `json:"receiverHandle"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Fetch enriched interactions
	interactions, err := c.InteractionService.GetInteractionsByReceiverHandle(context.TODO(), request.ReceiverHandle)
	if err != nil {
		http.Error(w, `{"error": "Failed to fetch interactions"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(interactions)
}
