package routes

import (
	"vibin_server/controllers"
	"vibin_server/services"

	"github.com/gorilla/mux"
)

// RegisterChatRoutes registers chat-related endpoints
func RegisterChatRoutes(r *mux.Router, chatService *services.ChatService) {
	controller := controllers.NewChatController(chatService)

	chatRouter := r.PathPrefix("/api/chat").Subrouter()

	// ✅ Fetch messages
	chatRouter.HandleFunc("/messages", controller.HandleGetMessages).Methods("GET")

	// ✅ Mark messages as read (Updated to include userHandle)
	chatRouter.HandleFunc("/messages/mark-as-read", controller.HandleMarkMessagesAsRead).Methods("POST")
}
