package controllers

import (
	"net/http"
	"vibin_server/helpers"
	"vibin_server/services"

	"github.com/gorilla/mux"
)

// HealthCheckHandler provides a basic health check
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	helpers.WriteJSONResponse(w, http.StatusOK, map[string]string{"message": "Server is running!"})
}

// WelcomeHandler provides a welcome message
func WelcomeHandler(w http.ResponseWriter, r *http.Request) {
	helpers.WriteJSONResponse(w, http.StatusOK, map[string]string{"message": "Welcome to the server! This is the Vibin API."})
}

// AddMessageHandler handles adding a message
func AddMessageHandler(w http.ResponseWriter, r *http.Request) {
	message := services.MockAddMessage()
	helpers.WriteJSONResponse(w, http.StatusOK, message)
}

// GetMessagesHandler handles fetching messages
func GetMessagesHandler(w http.ResponseWriter, r *http.Request) {
	matchID := mux.Vars(r)["matchId"]
	messages := services.MockGetMessages(matchID)
	helpers.WriteJSONResponse(w, http.StatusOK, messages)
}

// AddUserProfileHandler handles adding a user profile
func AddUserProfileHandler(w http.ResponseWriter, r *http.Request) {
	profile := services.MockAddUserProfile()
	helpers.WriteJSONResponse(w, http.StatusOK, profile)
}

// GetUserProfileHandler handles fetching a user profile
func GetUserProfileHandler(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userId"]
	profile := services.MockGetUserProfile(userID)
	helpers.WriteJSONResponse(w, http.StatusOK, profile)
}

// DeleteUserProfileHandler handles deleting a user profile
func DeleteUserProfileHandler(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userId"]
	services.MockDeleteUserProfile(userID)
	helpers.WriteJSONResponse(w, http.StatusOK, map[string]string{"message": "Profile deleted successfully", "userId": userID})
}
