package routes

import (
	"vibin_server/controllers"
	"vibin_server/services"

	"github.com/gorilla/mux"
)

// RegisterMatchRoutes sets up routes for match-related operations under /api/match
func RegisterMatchRoutes(r *mux.Router, matchService *services.MatchService) {
	// Initialize the controller with the ActionService
	controller := controllers.NewMatchController(matchService)

	// Create a subrouter for /api/match
	matchRouter := r.PathPrefix("/api/match").Subrouter()

	// Define routes and their corresponding handlers
	matchRouter.HandleFunc("/connections", controller.GetConnections).Methods("GET")
	matchRouter.HandleFunc("", controller.GetFilteredProfiles).Methods("GET") // Handles /api/match with query parameters
	matchRouter.HandleFunc("/newLikes", controller.GetNewLikes).Methods("GET")
	matchRouter.HandleFunc("/pings", controller.GetPings).Methods("GET")
}
