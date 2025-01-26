package routes

import (
	"vibin_server/controllers"
	"vibin_server/services"

	"github.com/gorilla/mux"
)

// RegisterActionRoutes sets up routes for action-related operations
func RegisterActionRoutes(r *mux.Router, actionService *services.ActionService) {
	// Initialize the controller with the ActionService
	controller := controllers.NewActionController(actionService)

	r.HandleFunc("/pingAction", controller.HandlePingAction).Methods("POST")
	r.HandleFunc("/action", controller.HandleAction).Methods("POST")
	r.HandleFunc("/currentMatches", controller.GetCurrentMatches).Methods("GET")
	r.HandleFunc("/newLikes", controller.GetNewLikes).Methods("GET")
	r.HandleFunc("/pings", controller.GetPings).Methods("GET")
	r.HandleFunc("/filteredProfiles", controller.GetFilteredProfiles).Methods("GET")
}
