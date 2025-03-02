package controllers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
	"vibin_server/models"
	"vibin_server/services"

	"github.com/google/uuid"
)

// GroupChatController struct
type GroupChatController struct {
	GroupChatService *services.GroupChatService
}

// NewGroupChatController initializes the group chat controller
func NewGroupChatController(service *services.GroupChatService) *GroupChatController {
	return &GroupChatController{GroupChatService: service}
}

// HandleCreateGroupMessage - Handles sending a new group message
func (c *GroupChatController) HandleCreateGroupMessage(w http.ResponseWriter, r *http.Request) {
	var request struct {
		GroupID  string   `json:"groupId"`
		SenderID string   `json:"senderId"`
		Content  string   `json:"content"`
		ImageURL *string  `json:"imageUrl,omitempty"`
		Members  []string `json:"members"`
	}

	// Decode request body
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	// ✅ Validate required fields
	if request.GroupID == "" || request.SenderID == "" || request.Content == "" {
		http.Error(w, `{"error": "Missing required fields: groupId, senderId, or content"}`, http.StatusBadRequest)
		return
	}

	// ✅ Generate a unique message ID
	messageID := uuid.New().String()

	// ✅ Set createdAt timestamp
	createdAt := time.Now().Format(time.RFC3339)

	// ✅ Initialize group message struct
	message := models.GroupMessage{
		GroupID:     request.GroupID,
		CreatedAt:   createdAt,
		MessageID:   messageID,
		SenderID:    request.SenderID,
		Content:     request.Content,
		ImageURL:    request.ImageURL,
		IsRead:      make(map[string]bool),
		Likes:       make(map[string]bool),
		ReadCount:   1, // Sender has read the message
		LikeCount:   0,
		MemberCount: len(request.Members),
	}

	// ✅ Initialize isRead map (Only sender has read the message initially)
	for _, member := range request.Members {
		message.IsRead[member] = false
	}
	message.IsRead[request.SenderID] = true // Sender has read their own message

	log.Printf("📩 Creating group message: %+v", message)

	// ✅ Save message to DynamoDB using GroupChatService
	err := c.GroupChatService.CreateGroupMessage(context.TODO(), message)
	if err != nil {
		log.Printf("❌ Failed to send group message: %v", err)
		http.Error(w, `{"error": "Failed to send group message"}`, http.StatusInternalServerError)
		return
	}

	// ✅ Send success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Group message sent successfully",
	})
}
