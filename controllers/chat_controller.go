package controllers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"
	"vibin_server/models"
	"vibin_server/services"

	"github.com/google/uuid"
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

// HandleSendMessage - Handles sending a new message
func (c *ChatController) HandleSendMessage(w http.ResponseWriter, r *http.Request) {
	var message models.Message

	// Decode request body
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	// ✅ Validate required fields
	if message.MatchID == "" || message.SenderID == "" || message.Content == "" {
		http.Error(w, `{"error": "Missing required fields: matchId, senderId, or content"}`, http.StatusBadRequest)
		return
	}

	// ✅ Generate a unique message ID if not provided
	if message.MessageID == "" {
		message.MessageID = uuid.New().String()
	}

	// ✅ Set createdAt timestamp
	message.CreatedAt = time.Now().Format(time.RFC3339)

	// ✅ Set `isUnread` to "true" by default
	message.SetIsUnread(true)

	log.Printf("📩 Received message request: %+v", message)

	// ✅ Save message to DynamoDB using the existing SendMessage function
	err := c.ChatService.SendMessage(context.TODO(), message)
	if err != nil {
		log.Printf("❌ Failed to send message: %v", err)
		http.Error(w, `{"error": "Failed to send message"}`, http.StatusInternalServerError)
		return
	}

	// ✅ Send success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Message sent successfully",
	})
}

func (c *ChatController) HandleLikeMessage(w http.ResponseWriter, r *http.Request) {
	var request struct {
		MatchID   string `json:"matchId"`
		CreatedAt string `json:"createdAt"` // ✅ Use `createdAt` instead of `messageId`
		Liked     bool   `json:"liked"`
	}

	// Decode request body
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	// ✅ Validate required fields
	if request.MatchID == "" || request.CreatedAt == "" {
		http.Error(w, `{"error": "Missing required fields: matchId, createdAt"}`, http.StatusBadRequest)
		return
	}

	log.Printf("💖 Updating like status for message at %s in MatchID: %s to %v", request.CreatedAt, request.MatchID, request.Liked)

	// ✅ Call the service to update the like status
	err := c.ChatService.UpdateMessageLikeStatus(context.TODO(), request.MatchID, request.CreatedAt, request.Liked)
	if err != nil {
		log.Printf("❌ Failed to update like status: %v", err)
		http.Error(w, `{"error": "Failed to update like status"}`, http.StatusInternalServerError)
		return
	}

	// ✅ Send success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Like status updated successfully",
	})
}
