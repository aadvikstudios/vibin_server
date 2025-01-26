package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"vibin_server/services"
)

// ActionController handles HTTP requests for actions
type MatchController struct {
	MatchService *services.MatchService
}

// NewActionController creates a new ActionController instance
func NewMatchController(matchService *services.MatchService) *MatchController {
	return &MatchController{MatchService: matchService}
}

// GetFilteredProfiles handles fetching filtered profiles
func (ac *MatchController) GetFilteredProfiles(w http.ResponseWriter, r *http.Request) {
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

// GetPings handles fetching pings for a user
func (ac *MatchController) GetPings(w http.ResponseWriter, r *http.Request) {
	emailId := r.URL.Query().Get("emailId")
	if emailId == "" {
		http.Error(w, "userId is required", http.StatusBadRequest)
		return
	}

	pings, err := ac.MatchService.GetPings(context.Background(), emailId)
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
func (ac *MatchController) GetCurrentMatches(w http.ResponseWriter, r *http.Request) {
	emailId := r.URL.Query().Get("emailId")
	if emailId == "" {
		http.Error(w, "userId is required", http.StatusBadRequest)
		return
	}

	matches, err := ac.MatchService.GetCurrentMatches(context.Background(), emailId)
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
func (ac *MatchController) GetNewLikes(w http.ResponseWriter, r *http.Request) {
	emailId := r.URL.Query().Get("emailId")
	if emailId == "" {
		http.Error(w, "userId is required", http.StatusBadRequest)
		return
	}

	likes, err := ac.MatchService.GetNewLikes(context.Background(), emailId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"likes": likes,
	})
}
