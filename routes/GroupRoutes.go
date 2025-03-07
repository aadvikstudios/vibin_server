package routes

import (
	"vibin_server/controllers"
	"vibin_server/services"

	"github.com/gorilla/mux"
)

// RegisterGroupChatRoutes registers group chat-related routes
func RegisterGroupChatRoutes(r *mux.Router, groupChatService *services.GroupChatService) {
	controller := controllers.NewGroupChatController(groupChatService)

	groupRouter := r.PathPrefix("/api/groupchat").Subrouter()
	groupRouter.HandleFunc("/message", controller.HandleCreateGroupMessage).Methods("POST") // ✅ Create a new group message
	groupRouter.HandleFunc("/messages", controller.HandleGetGroupMessages).Methods("GET")   // ✅ Fetch group messages

}
