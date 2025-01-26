package routes

import (
	"vibin_server/controllers"

	"github.com/gorilla/mux"
)

// RegisterRoutes sets up the routes for the application
func RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/health", controllers.HealthCheckHandler).Methods("GET")
	r.HandleFunc("/welcome", controllers.WelcomeHandler).Methods("GET")
	r.HandleFunc("/", controllers.WelcomeHandler).Methods("GET")
	r.HandleFunc("/messages", controllers.AddMessageHandler).Methods("POST")
	r.HandleFunc("/messages/{matchId}", controllers.GetMessagesHandler).Methods("GET")
	r.HandleFunc("/profiles", controllers.AddUserProfileHandler).Methods("POST")
	r.HandleFunc("/profiles/{userId}", controllers.GetUserProfileHandler).Methods("GET")
	r.HandleFunc("/profiles/{userId}", controllers.DeleteUserProfileHandler).Methods("DELETE")
}
