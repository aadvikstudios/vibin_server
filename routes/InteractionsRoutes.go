package routes

import (
	"vibin_server/controllers"
	"vibin_server/services"

	"github.com/gorilla/mux"
)

// RegisterInteractionsRoutes registers all interaction-related routes under `/api/interactions`
func RegisterInteractionsRoutes(router *mux.Router, interactionService *services.InteractionService) {
	controller := &controllers.InteractionController{InteractionService: interactionService}

	interactionRouter := router.PathPrefix("/api/interactions").Subrouter()

	// Interaction Routes
	interactionRouter.HandleFunc("", controller.CreateInteractionHandler).Methods("POST")       // ✅ Create or update interactions (like, ping, approve, reject)
	interactionRouter.HandleFunc("", controller.GetUserInteractionsHandler).Methods("GET")      // ✅ Get all interactions for a user
	interactionRouter.HandleFunc("/matches", controller.GetMutualMatchesHandler).Methods("GET") // ✅ Get mutual matches
}
