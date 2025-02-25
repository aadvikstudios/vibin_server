package controllers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
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

// GetUserProfileByEmail fetches a user profile using the email ID from the GSI
func (c *UserProfileController) GetUserProfileByEmail(w http.ResponseWriter, r *http.Request) {
	var request struct {
		EmailID string `json:"emailId"`
	}

	// Decode JSON request
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.EmailID == "" {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Fetch user profile
	profile, err := c.UserProfileService.GetUserProfileByEmail(context.TODO(), request.EmailID)
	if err != nil {
		http.Error(w, "Failed to fetch profile", http.StatusInternalServerError)
		return
	}

	// If profile is not found, return 404
	if profile == nil {
		http.Error(w, "Profile not found", http.StatusNotFound)
		return
	}

	// Return user profile
	json.NewEncoder(w).Encode(profile)
}

func (c *UserProfileController) CheckUserHandleAvailability(w http.ResponseWriter, r *http.Request) {
	// Extract userhandle from query params
	userHandle := r.URL.Query().Get("userhandle")
	if userHandle == "" {
		http.Error(w, `{"error": "Missing required field: userhandle"}`, http.StatusBadRequest)
		return
	}

	log.Printf("🔍 API Request to check userhandle: %s", userHandle)

	// Use a context with timeout to avoid long-running requests
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if userhandle exists using `GetItem`
	isAvailable, err := c.UserProfileService.IsUserHandleAvailable(ctx, userHandle)
	if err != nil {
		log.Printf("❌ Internal Server Error while checking userhandle '%s': %v", userHandle, err)
		http.Error(w, `{"error": "Error checking userhandle"}`, http.StatusInternalServerError)
		return
	}

	// ✅ Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"available": isAvailable})
}

// CheckEmailAvailability checks if an email exists and returns `exists: true/false`
func (c *UserProfileController) CheckEmailAvailability(w http.ResponseWriter, r *http.Request) {
	var request struct {
		EmailID string `json:"emailId"`
	}

	// Decode JSON request
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.EmailID == "" {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Check if email exists
	exists, err := c.UserProfileService.CheckEmailExists(context.TODO(), request.EmailID)
	if err != nil {
		http.Error(w, "Error checking email availability", http.StatusInternalServerError)
		return
	}

	// Return response
	json.NewEncoder(w).Encode(map[string]bool{"exists": exists})
}

// GetUserHandleByEmail fetches the userhandle associated with an email
func (c *UserProfileController) GetUserHandleByEmail(w http.ResponseWriter, r *http.Request) {
	emailID := r.URL.Query().Get("emailId")
	if emailID == "" {
		http.Error(w, "Missing required parameter: emailId", http.StatusBadRequest)
		return
	}

	// Fetch userhandle
	userHandle, err := c.UserProfileService.GetUserHandleByEmail(context.TODO(), emailID)
	if err != nil {
		http.Error(w, "Error fetching userhandle", http.StatusInternalServerError)
		return
	}

	// If userhandle is empty, return 404
	if userHandle == "" {
		http.Error(w, "Email not found", http.StatusNotFound)
		return
	}

	// Return userhandle
	json.NewEncoder(w).Encode(map[string]string{"userhandle": userHandle})
}
