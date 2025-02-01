package routes

import (
	"vibin_server/controllers"
	"vibin_server/services"

	"github.com/gorilla/mux"
)

// RegisterActionRoutes sets up routes for action-related operations under /api/action
func RegisterActionRoutes(r *mux.Router, actionService *services.ActionService) {
	// Initialize the controller with the ActionService
	controller := controllers.NewActionController(actionService)

	// Create a subrouter for /api/action
	actionRouter := r.PathPrefix("/api/action").Subrouter()

	// Define routes and their corresponding handlers
	actionRouter.HandleFunc("/sendPing", controller.HandleSendPing).Methods("POST")
	// actionRouter.HandleFunc("/pingActionGet", controller.HandlePingAction).Methods("POST")
	actionRouter.HandleFunc("/action", controller.HandleAction).Methods("POST")
}
