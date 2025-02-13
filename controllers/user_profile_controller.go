package controllers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"vibin_server/models"
	"vibin_server/services"

	"github.com/gorilla/mux"
)

// UserProfileController handles requests related to user profiles
type UserProfileController struct {
	UserProfileService *services.UserProfileService
}

// NewUserProfileController creates a new instance of UserProfileController
func NewUserProfileController(userProfileService *services.UserProfileService) *UserProfileController {
	return &UserProfileController{UserProfileService: userProfileService}
}

func (c *UserProfileController) CreateUserProfile(w http.ResponseWriter, r *http.Request) {
	log.Println("CreateUserProfile called...")

	var profile models.UserProfile

	// Decode the request body
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		log.Printf("Failed to decode request body: %v\n", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	log.Printf("Request payload before generating userId: %+v\n", profile)

	// Call the service to add the user profileu
	createdProfile, err := c.UserProfileService.AddUserProfile(context.TODO(), profile)
	if err != nil {
		log.Printf("Failed to add profile: %v\n", err)
		http.Error(w, "Failed to add profile", http.StatusInternalServerError)
		return
	}

	// Return the created profile
	log.Printf("Profile added successfully: %+v\n", createdProfile)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Profile added successfully",
		"profile": createdProfile,
	})
}

// UpdateUserProfile handles updating an existing user profile
func (c *UserProfileController) UpdateUserProfile(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userId"]
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	updatedProfile, err := c.UserProfileService.UpdateUserProfile(context.TODO(), userID, updates)
	if err != nil {
		http.Error(w, "Failed to update profile", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Profile updated successfully",
		"profile": updatedProfile,
	})
}

// GetUserProfileByID handles fetching a user profile by ID
func (c *UserProfileController) GetUserProfileByID(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userId"]

	profile, err := c.UserProfileService.GetUserProfile(context.TODO(), userID)
	if err != nil {
		http.Error(w, "Failed to fetch profile", http.StatusInternalServerError)
		return
	}

	if profile == nil {
		http.Error(w, "Profile not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(profile)
}

// GetUserProfileByEmail handles fetching a user profile by email
func (c *UserProfileController) GetUserProfileByEmail(w http.ResponseWriter, r *http.Request) {
	emailID := mux.Vars(r)["emailId"]
	targetEmailId, exists := mux.Vars(r)["targetEmailId"] // Check if targetEmailId exists

	// Convert targetEmailId to pointer if it exists, otherwise pass nil
	var targetEmailPtr *string
	if exists && targetEmailId != "" {
		targetEmailPtr = &targetEmailId
	}

	// Fetch the user profile, dynamically computing distance if targetEmailPtr is provided
	profile, err := c.UserProfileService.GetUserProfileByEmail(context.TODO(), emailID, targetEmailPtr)
	if err != nil {
		http.Error(w, "Failed to fetch profile by email", http.StatusInternalServerError)
		return
	}

	if profile == nil {
		http.Error(w, "Profile not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(profile)
}

// DeleteUserProfile handles deleting a user profile
func (c *UserProfileController) DeleteUserProfile(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userId"]

	if err := c.UserProfileService.DeleteUserProfile(context.TODO(), userID); err != nil {
		http.Error(w, "Failed to delete profile", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Profile deleted successfully",
	})
}
