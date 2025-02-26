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

// HandleGetMessages - Fetch messages based on matchId
func (c *ChatController) HandleGetMessages(w http.ResponseWriter, r *http.Request) {
	// ✅ Parse query parameters
	matchID := r.URL.Query().Get("matchId")
	limitStr := r.URL.Query().Get("limit")

	// ✅ Validate matchId
	if matchID == "" {
		http.Error(w, `{"error": "matchId is required"}`, http.StatusBadRequest)
		return
	}

	// ✅ Convert limit from string to int (default: 50)
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50 // Default to 50 messages
	}

	log.Printf("🔍 Fetching messages for matchId: %s, Limit: %d", matchID, limit)

	// ✅ Fetch messages
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

// ✅ HandleMarkMessagesAsRead - Mark messages received by user as read
func (c *ChatController) HandleMarkMessagesAsRead(w http.ResponseWriter, r *http.Request) {
	var request struct {
		MatchID    string `json:"matchId"`
		UserHandle string `json:"userHandle"` // ✅ Who is marking messages as read
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	log.Printf("🔄 Marking messages as read for matchId: %s, User: %s", request.MatchID, request.UserHandle)

	// ✅ Call service function to update messages
	err := c.ChatService.MarkMessagesAsRead(context.TODO(), request.MatchID, request.UserHandle)
	if err != nil {
		http.Error(w, `{"error": "Failed to mark messages as read"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Messages received by user marked as read"})
}
