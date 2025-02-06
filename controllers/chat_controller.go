package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"vibin_server/services"
)

// ChatController handles HTTP requests for chat operations
type ChatController struct {
	ChatService *services.ChatService
}

// NewChatController creates a new ChatController instance
func NewChatController(chatService *services.ChatService) *ChatController {
	return &ChatController{ChatService: chatService}
}

// CreateMessage handles adding a new message with text and/or an image
func (cc *ChatController) CreateMessage(w http.ResponseWriter, r *http.Request) {
	var message services.Message

	// Decode JSON request body
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		fmt.Printf("[ERROR] CreateMessage: Invalid request payload: %v\n", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Debug: Log the received message
	fmt.Printf("[DEBUG] CreateMessage: Received request: %+v\n", message)

	// Validate required fields
	if message.MatchID == "" || message.SenderID == "" {
		fmt.Println("[ERROR] CreateMessage: Missing required fields")
		http.Error(w, "Missing required fields: matchId and senderId", http.StatusBadRequest)
		return
	}

	// Ensure at least one of content or imageUrl is provided
	if message.Content == "" && message.ImageURL == "" {
		fmt.Println("[ERROR] CreateMessage: Either content or imageUrl must be provided")
		http.Error(w, "Either content or imageUrl must be provided", http.StatusBadRequest)
		return
	}

	// Save message to database
	err := cc.ChatService.SaveMessage(message)
	if err != nil {
		fmt.Printf("[ERROR] CreateMessage: Failed to save message: %v\n", err)
		http.Error(w, "Failed to save message", http.StatusInternalServerError)
		return
	}

	// Respond with success
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message":   "Message saved successfully",
		"messageId": message.MessageID,
	})
}

// GetMessagesByMatchID retrieves messages for a specific match ID
func (cc *ChatController) GetMessagesByMatchID(w http.ResponseWriter, r *http.Request) {
	matchID := r.URL.Query().Get("matchId")
	if matchID == "" {
		log.Println("[DEBUG] GetMessagesByMatchID: matchId is missing in the request")
		http.Error(w, "matchId is required", http.StatusBadRequest)
		return
	}
	log.Printf("[DEBUG] GetMessagesByMatchID: Received request for matchId: %s\n", matchID)

	messages, err := cc.ChatService.GetMessagesByMatchID(matchID)
	if err != nil {
		log.Printf("[ERROR] GetMessagesByMatchID: Failed to fetch messages for matchId %s: %v\n", matchID, err)
		http.Error(w, "Failed to fetch messages", http.StatusInternalServerError)
		return
	}

	log.Printf("[DEBUG] GetMessagesByMatchID: Successfully fetched messages: %+v\n", messages)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(messages); err != nil {
		log.Printf("[ERROR] GetMessagesByMatchID: Failed to encode response: %v\n", err)
		http.Error(w, "Failed to send response", http.StatusInternalServerError)
		return
	}
}

// MarkMessagesAsRead handles marking messages as read
func (cc *ChatController) MarkMessagesAsRead(w http.ResponseWriter, r *http.Request) {
	var request struct {
		MatchID string `json:"matchId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Printf("[DEBUG] MarkMessagesAsRead: Invalid request payload: %v\n", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	log.Printf("[DEBUG] MarkMessagesAsRead: Received request to mark messages as read for matchId: %s\n", request.MatchID)

	if request.MatchID == "" {
		log.Println("[DEBUG] MarkMessagesAsRead: matchId is missing in the request payload")
		http.Error(w, "matchId is required", http.StatusBadRequest)
		return
	}

	err := cc.ChatService.MarkMessagesAsRead(request.MatchID)
	if err != nil {
		log.Printf("[ERROR] MarkMessagesAsRead: Failed to mark messages as read for matchId %s: %v\n", request.MatchID, err)
		http.Error(w, "Failed to mark messages as read", http.StatusInternalServerError)
		return
	}

	log.Printf("[DEBUG] MarkMessagesAsRead: Successfully marked messages as read for matchId: %s\n", request.MatchID)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "Messages marked as read"}); err != nil {
		log.Printf("[ERROR] MarkMessagesAsRead: Failed to encode response: %v\n", err)
		http.Error(w, "Failed to send response", http.StatusInternalServerError)
		return
	}
}

// LikeMessage handles liking a message
func (cc *ChatController) LikeMessage(w http.ResponseWriter, r *http.Request) {
	var request struct {
		MatchID   string `json:"matchId"`
		MessageID string `json:"messageId"`
		CreatedAt string `json:"createdAt"`
		Liked     bool   `json:"liked"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	err := cc.ChatService.LikeMessage(request.MatchID, request.MessageID, request.CreatedAt, request.Liked)
	if err != nil {
		http.Error(w, "Failed to like message", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Message liked successfully"})
}
