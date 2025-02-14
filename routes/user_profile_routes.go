package routes

import (
	"vibin_server/controllers"
	"vibin_server/services"

	"github.com/gorilla/mux"
)

// RegisterUserProfileRoutes sets up routes for user profile operations under /api/profiles
func RegisterUserProfileRoutes(r *mux.Router, userProfileService *services.UserProfileService) {
	// Initialize the controller with the provided UserProfileService
	controller := controllers.NewUserProfileController(userProfileService)

	profileRouter := r.PathPrefix("/api/profiles").Subrouter()
	profileRouter.HandleFunc("", controller.CreateUserProfile).Methods("POST")
	profileRouter.HandleFunc("/email/profile", controller.GetUserProfileByEmail).Methods("POST")
	profileRouter.HandleFunc("/clear-interactions", controller.ClearUserInteractions).Methods("POST")
}
