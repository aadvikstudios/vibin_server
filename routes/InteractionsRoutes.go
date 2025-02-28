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
	interactionRouter.HandleFunc("", controller.CreateInteractionHandler).Methods("POST")               // ✅ Create or update interactions (like, ping, approve, reject)
	interactionRouter.HandleFunc("/sent", controller.GetSentInteractionsHandler).Methods("GET")         // ✅ Get interactions initiated by the user
	interactionRouter.HandleFunc("/received", controller.GetReceivedInteractionsHandler).Methods("GET") // ✅ Get interactions received by the user
	interactionRouter.HandleFunc("/matches", controller.GetMutualMatchesHandler).Methods("GET")         // ✅ Get mutual matches
}
