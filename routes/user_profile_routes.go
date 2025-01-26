package routes

import (
	"vibin_server/controllers"
	"vibin_server/services"

	"github.com/gorilla/mux"
)

// RegisterUserProfileRoutes sets up routes for user profile operations
func RegisterUserProfileRoutes(r *mux.Router, userProfileService *services.UserProfileService) {
	// Initialize the controller with the provided UserProfileService
	controller := controllers.NewUserProfileController(userProfileService)

	// Define routes and their corresponding handlers
	r.HandleFunc("/profiles", controller.CreateUserProfile).Methods("POST")
	r.HandleFunc("/profiles/{userId}", controller.UpdateUserProfile).Methods("PATCH")
	r.HandleFunc("/profiles/{userId}", controller.GetUserProfileByID).Methods("GET")
	r.HandleFunc("/profiles/email/{emailId}", controller.GetUserProfileByEmail).Methods("GET")
	r.HandleFunc("/profiles/{userId}", controller.DeleteUserProfile).Methods("DELETE")
}
