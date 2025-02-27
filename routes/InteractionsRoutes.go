package routes

import (
	"vibin_server/controllers"
	"vibin_server/services"

	"github.com/gorilla/mux"
)

// RegisterPendingInviteRoutes registers all invite-related routes under `/api/invites`
func RegisterInteractionsRoutes(router *mux.Router, interactionService *services.InteractionService) {
	controller := &controllers.InteractionController{InteractionService: interactionService}

	interactionRouter := router.PathPrefix("/api/interactions").Subrouter()

	// Interaction Routes
	interactionRouter.HandleFunc("", controller.CreateInteractionHandler).Methods("POST")
	interactionRouter.HandleFunc("", controller.GetUserInteractionsHandler).Methods("GET")
}
