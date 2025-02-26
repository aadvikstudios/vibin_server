package routes

import (
	"vibin_server/controllers"
	"vibin_server/services"

	"github.com/gorilla/mux"
)

// RegisterChatRoutes registers chat-related API endpoints
func RegisterChatRoutes(r *mux.Router, chatService *services.ChatService) {
	controller := controllers.NewChatController(chatService)

	chatRouter := r.PathPrefix("/api/chat").Subrouter()
	chatRouter.HandleFunc("/messages", controller.HandleGetMessages).Methods("GET") // ✅ Fetch messages by matchId
}
