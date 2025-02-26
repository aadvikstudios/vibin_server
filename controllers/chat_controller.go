package controllers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"vibin_server/services"
)

// ChatController struct
type ChatController struct {
	ChatService *services.ChatService
}

// NewChatController initializes the chat controller
func NewChatController(service *services.ChatService) *ChatController {
	return &ChatController{ChatService: service}
}

// HandleGetMessages - Fetch messages by matchId
func (c *ChatController) HandleGetMessages(w http.ResponseWriter, r *http.Request) {
	// ✅ Parse query parameters
	matchID := r.URL.Query().Get("matchId")
	limitStr := r.URL.Query().Get("limit")

	if matchID == "" {
		http.Error(w, `{"error": "matchId is required"}`, http.StatusBadRequest)
		return
	}

	// ✅ Convert limit to integer (default to 50 if not provided)
	limit := 50
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			limit = parsedLimit
		}
	}

	log.Printf("🔍 Fetching messages for matchId: %s, Limit: %d", matchID, limit)

	// ✅ Fetch messages from DynamoDB
	messages, err := c.ChatService.GetMessagesByMatchID(context.TODO(), matchID, limit)
	if err != nil {
		log.Printf("❌ Error fetching messages: %v", err)
		http.Error(w, `{"error": "Failed to fetch messages"}`, http.StatusInternalServerError)
		return
	}

	// ✅ Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}
