package routes

import (
	"vibin_server/controllers"
	"vibin_server/services"

	"github.com/gorilla/mux"
)

// RegisterChatRoutes sets up routes for chat-related operations under /api/chat
func RegisterChatRoutes(r *mux.Router, chatService *services.ChatService) {
	// Initialize the controller with the ChatService
	controller := controllers.NewChatController(chatService)

	// Create a subrouter for /api/chat
	chatRouter := r.PathPrefix("/api/chat").Subrouter()

	// Define routes and their corresponding handlers
	chatRouter.HandleFunc("/message", controller.CreateMessage).Methods("POST")
	chatRouter.HandleFunc("/messages", controller.GetMessagesByMatchID).Methods("GET")
	chatRouter.HandleFunc("/messages/mark-as-read", controller.MarkMessagesAsRead).Methods("POST")
	chatRouter.HandleFunc("/messages/like", controller.LikeMessage).Methods("PATCH")
}
