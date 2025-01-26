package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"vibin_server/services"
)

// ActionController handles HTTP requests for actions
type ActionController struct {
	ActionService *services.ActionService
}

// NewActionController creates a new ActionController instance
func NewActionController(actionService *services.ActionService) *ActionController {
	return &ActionController{ActionService: actionService}
}

// HandlePingAction processes ping actions
func (ac *ActionController) HandlePingAction(w http.ResponseWriter, r *http.Request) {
	var request struct {
		UserID       string `json:"userId"`
		TargetUserID string `json:"targetUserId"`
		Action       string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	err := ac.ActionService.PingAction(context.Background(), request.UserID, request.TargetUserID, request.Action)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Ping action processed successfully"})
}

// GetFilteredProfiles handles fetching filtered profiles
func (ac *ActionController) GetFilteredProfiles(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	gender := r.URL.Query().Get("gender")

	if userID == "" || gender == "" {
		http.Error(w, "userId and gender are required", http.StatusBadRequest)
		return
	}

	// Logic to get filtered profiles
	// ...

	json.NewEncoder(w).Encode(map[string]interface{}{"profiles": []string{}})
}

// HandleAction processes user actions such as "liked", "notliked", and "pinged"
func (ac *ActionController) HandleAction(w http.ResponseWriter, r *http.Request) {
	var request struct {
		UserID       string `json:"userId"`
		TargetUserID string `json:"targetUserId"`
		Action       string `json:"action"`
		PingNote     string `json:"pingNote"`
	}

	// Decode the request payload
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if request.UserID == "" || request.TargetUserID == "" || request.Action == "" {
		http.Error(w, "userId, targetUserId, and action are required", http.StatusBadRequest)
		return
	}

	// Process the action
	response, err := ac.ActionService.ProcessAction(context.Background(), request.UserID, request.TargetUserID, request.Action, request.PingNote)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Send a successful response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetPings handles fetching pings for a user
func (ac *ActionController) GetPings(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		http.Error(w, "userId is required", http.StatusBadRequest)
		return
	}

	pings, err := ac.ActionService.GetPings(context.Background(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Pings fetched successfully",
		"pings":   pings,
	})
}

// GetCurrentMatches handles fetching current matches for a user
func (ac *ActionController) GetCurrentMatches(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		http.Error(w, "userId is required", http.StatusBadRequest)
		return
	}

	matches, err := ac.ActionService.GetCurrentMatches(context.Background(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"matches": matches,
	})
}

// GetNewLikes handles fetching new likes for a user
func (ac *ActionController) GetNewLikes(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		http.Error(w, "userId is required", http.StatusBadRequest)
		return
	}

	likes, err := ac.ActionService.GetNewLikes(context.Background(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"likes": likes,
	})
}

// Other handler functions...
