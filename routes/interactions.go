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
	interactionRouter.HandleFunc("/get", controller.HandleGetInteractions).Methods("POST")
}
