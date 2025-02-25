package controllers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"vibin_server/models"
	"vibin_server/services"
)

// UserProfileController handles user profile-related operations
type UserProfileController struct {
	UserProfileService *services.UserProfileService
}

// NewUserProfileController creates a new instance of UserProfileController
func NewUserProfileController(userProfileService *services.UserProfileService) *UserProfileController {
	return &UserProfileController{UserProfileService: userProfileService}
}

func (c *UserProfileController) CreateUserProfile(w http.ResponseWriter, r *http.Request) {
	var profile models.UserProfile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	createdProfile, err := c.UserProfileService.AddUserProfile(context.TODO(), profile)
	if err != nil {
		http.Error(w, "Failed to add profile", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(createdProfile)
}

func (c *UserProfileController) GetUserProfileByEmail(w http.ResponseWriter, r *http.Request) {
	var request struct {
		EmailID       string  `json:"emailId"`
		TargetEmailID *string `json:"targetEmailId,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	profile, err := c.UserProfileService.GetUserProfileByEmail(context.TODO(), request.EmailID, request.TargetEmailID)
	if err != nil {
		http.Error(w, "Failed to fetch profile", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(profile)
}

// CheckUserHandleAvailability checks if the given user handle is unique using the GSI
func (c *UserProfileController) CheckUserHandleAvailability(w http.ResponseWriter, r *http.Request) {
	// Extract userhandle from query params
	userHandle := r.URL.Query().Get("userhandle")
	if userHandle == "" {
		http.Error(w, "Missing required field: userhandle", http.StatusBadRequest)
		return
	}

	log.Printf("🔍 Checking availability for userhandle: %s", userHandle)

	// Call service to check uniqueness
	isAvailable, err := c.UserProfileService.IsUserHandleAvailable(context.TODO(), userHandle)
	if err != nil {
		log.Printf("❌ Error checking userhandle: %v", err)
		http.Error(w, "Error checking userhandle", http.StatusInternalServerError)
		return
	}

	// Respond based on availability
	if !isAvailable {
		log.Printf("❌ Userhandle %s is already taken.", userHandle)
		http.Error(w, "Userhandle is already taken", http.StatusConflict) // 409 Conflict
		return
	}

	log.Printf("✅ Userhandle %s is available.", userHandle)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Userhandle is available",
	})
}
