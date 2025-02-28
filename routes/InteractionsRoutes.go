package routes

import (
	"vibin_server/controllers"
	"vibin_server/services"

	"github.com/gorilla/mux"
)

func RegisterInteractionsRoutes(router *mux.Router, interactionService *services.InteractionService) {
	controller := &controllers.InteractionController{InteractionService: interactionService}

	interactionRouter := router.PathPrefix("/api/interactions").Subrouter()

	// Existing Routes
	interactionRouter.HandleFunc("", controller.CreateInteractionHandler).Methods("POST")
	interactionRouter.HandleFunc("/sent", controller.GetSentInteractionsHandler).Methods("GET")
	interactionRouter.HandleFunc("/received", controller.GetReceivedInteractionsHandler).Methods("GET")
	interactionRouter.HandleFunc("/matches", controller.GetMutualMatchesHandler).Methods("GET")

	// ✅ New Ping Handling Routes
	interactionRouter.HandleFunc("/ping/approve", controller.ApprovePingHandler).Methods("POST")
	interactionRouter.HandleFunc("/ping/decline", controller.DeclinePingHandler).Methods("POST")
}
