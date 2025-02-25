package routes

import (
	"vibin_server/controllers"
	"vibin_server/services"

	"github.com/gorilla/mux"
)

// RegisterInteractionRoutes sets up routes for interaction-related operations under /api/interactions
func RegisterInteractionRoutes(r *mux.Router, interactionService *services.InteractionService) {
	controller := controllers.NewInteractionController(interactionService)

	interactionRouter := r.PathPrefix("/api/interactions").Subrouter()
	interactionRouter.HandleFunc("/like", controller.HandleLikeUser).Methods("POST")
	interactionRouter.HandleFunc("/dislike", controller.HandleDislikeUser).Methods("POST")
}
