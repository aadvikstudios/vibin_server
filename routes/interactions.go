package routes

import (
	"vibin_server/controllers"
	"vibin_server/services"

	"github.com/gorilla/mux"
)

func RegisterInteractionRoutes(r *mux.Router, interactionService *services.InteractionService) {
	controller := controllers.NewInteractionController(interactionService)

	interactionRouter := r.PathPrefix("/api/interactions").Subrouter()
	interactionRouter.HandleFunc("/like", controller.HandleLikeUser).Methods("POST")
	interactionRouter.HandleFunc("/dislike", controller.HandleDislikeUser).Methods("POST")
	interactionRouter.HandleFunc("/ping", controller.HandlePingUser).Methods("POST")
	interactionRouter.HandleFunc("/ping/approve", controller.HandleApprovePing).Methods("POST") // ✅ Approve Ping
	interactionRouter.HandleFunc("/ping/decline", controller.HandleDeclinePing).Methods("POST") // ✅ Decline Ping
	interactionRouter.HandleFunc("/get", controller.HandleGetInteractions).Methods("POST")
}
