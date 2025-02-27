package routes

import (
	"vibin_server/controllers"
	"vibin_server/services"

	"github.com/gorilla/mux"
)

func RegisterMatchRoutes(r *mux.Router, matchService *services.MatchService) {
	controller := controllers.NewMatchController(matchService)

	matchRouter := r.PathPrefix("/api/match").Subrouter()
	matchRouter.HandleFunc("/get", controller.HandleGetMatches).Methods("POST") // ✅ Get matches based on userHandle
}
