package routes

import (
	"vibin_server/controllers"
	"vibin_server/services"

	"github.com/gorilla/mux"
)

func RegisterUserProfileRoutes(r *mux.Router, userProfileService *services.UserProfileService) {
	controller := controllers.NewUserProfileController(userProfileService)

	profileRouter := r.PathPrefix("/api/profile").Subrouter()
	profileRouter.HandleFunc("", controller.CreateUserProfile).Methods("POST")
	profileRouter.HandleFunc("/by-email", controller.GetUserProfileByEmail).Methods("POST")
	profileRouter.HandleFunc("/check-userhandle", controller.CheckUserHandleAvailability).Methods("GET") // ✅ New Route
	profileRouter.HandleFunc("/check-email", controller.CheckEmailAvailability).Methods("POST")
	profileRouter.HandleFunc("/fetch-userhandle", controller.GetUserHandleByEmail).Methods("GET")

}
