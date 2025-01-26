package controllers

import (
	"encoding/json"
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

// CreateMessage handles adding a new message
func (cc *ChatController) CreateMessage(w http.ResponseWriter, r *http.Request) {
	var message services.Message
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	err := cc.ChatService.SaveMessage(message)
	if err != nil {
		http.Error(w, "Failed to save message", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Message saved successfully"})
}

// GetMessagesByMatchID handles fetching messages by match ID
func (cc *ChatController) GetMessagesByMatchID(w http.ResponseWriter, r *http.Request) {
	matchID := r.URL.Query().Get("matchId")
	if matchID == "" {
		http.Error(w, "matchId is required", http.StatusBadRequest)
		return
	}

	messages, err := cc.ChatService.GetMessagesByMatchID(matchID)
	if err != nil {
		http.Error(w, "Failed to fetch messages", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(messages)
}

// MarkMessagesAsRead handles marking messages as read
func (cc *ChatController) MarkMessagesAsRead(w http.ResponseWriter, r *http.Request) {
	var request struct {
		MatchID string `json:"matchId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	err := cc.ChatService.MarkMessagesAsRead(request.MatchID)
	if err != nil {
		http.Error(w, "Failed to mark messages as read", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Messages marked as read"})
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
