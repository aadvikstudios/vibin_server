package routes

import (
	"vibin_server/controllers"
	"vibin_server/services"

	"github.com/gorilla/mux"
)

// RegisterChatRoutes sets up routes for chat-related operations
func RegisterChatRoutes(r *mux.Router, chatService *services.ChatService) {
	// Initialize the controller with the ChatService
	controller := controllers.NewChatController(chatService)

	r.HandleFunc("/message", controller.CreateMessage).Methods("POST")
	r.HandleFunc("/messages", controller.GetMessagesByMatchID).Methods("GET")
	r.HandleFunc("/messages/mark-as-read", controller.MarkMessagesAsRead).Methods("POST")
	r.HandleFunc("/messages/like", controller.LikeMessage).Methods("PATCH")
}
