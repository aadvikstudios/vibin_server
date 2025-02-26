package routes

import (
	"vibin_server/controllers"
	"vibin_server/services"

	"github.com/gorilla/mux"
)

// RegisterChatRoutes registers chat-related routes
func RegisterChatRoutes(r *mux.Router, chatService *services.ChatService) {
	controller := controllers.NewChatController(chatService)

	chatRouter := r.PathPrefix("/api/chat").Subrouter()
	chatRouter.HandleFunc("/message", controller.HandleSendMessage).Methods("POST")                      // ✅ Send message
	chatRouter.HandleFunc("/messages", controller.HandleGetMessages).Methods("GET")                      // ✅ Get messages
	chatRouter.HandleFunc("/messages/mark-as-read", controller.HandleMarkMessagesAsRead).Methods("POST") // ✅ Mark messages as read
}
