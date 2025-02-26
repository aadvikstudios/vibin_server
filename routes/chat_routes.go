package routes

import (
	"vibin_server/controllers"

	"github.com/gorilla/mux"
)

// RegisterChatRoutes registers chat-related endpoints
func RegisterChatRoutes(r *mux.Router, chatController *controllers.ChatController) {
	chatRouter := r.PathPrefix("/api/chat").Subrouter()

	// ✅ Fetch messages
	chatRouter.HandleFunc("/messages", chatController.HandleGetMessages).Methods("GET")

	// ✅ Mark messages as read (Updated to include userHandle)
	chatRouter.HandleFunc("/messages/mark-as-read", chatController.HandleMarkMessagesAsRead).Methods("POST")
}
