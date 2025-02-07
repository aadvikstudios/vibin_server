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

	// Create a subrouter for the /api/profiles base path
	profileRouter := r.PathPrefix("/api/profiles").Subrouter()

	// Define routes and their corresponding handlers
	profileRouter.HandleFunc("", controller.CreateUserProfile).Methods("POST")
	// profileRouter.HandleFunc("/{userId}", controller.UpdateUserProfile).Methods("PATCH")
	// profileRouter.HandleFunc("/{userId}", controller.GetUserProfileByID).Methods("GET")
	profileRouter.HandleFunc("/email/{emailId}", controller.GetUserProfileByEmail).Methods("GET")
	// profileRouter.HandleFunc("/{userId}", controller.DeleteUserProfile).Methods("DELETE")
}
