package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"vibin_server/services"
)

// MatchController handles HTTP requests for match-related actions
type MatchController struct {
	MatchService *services.MatchService
}

// NewMatchController creates a new MatchController instance
func NewMatchController(matchService *services.MatchService) *MatchController {
	return &MatchController{MatchService: matchService}
}

// GetFilteredProfiles handles fetching filtered profiles based on dynamic criteria
func (ac *MatchController) GetFilteredProfiles(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	queryParams := r.URL.Query()
	emailId := queryParams.Get("emailId")
	gender := queryParams.Get("gender")

	if emailId == "" || gender == "" {
		http.Error(w, "emailId and gender are required", http.StatusBadRequest)
		return
	}

	// Prepare additional filters from query parameters
	additionalFilters := map[string]string{}
	for key, values := range queryParams {
		if key != "emailId" && key != "gender" {
			additionalFilters[key] = values[0]
		}
	}

	// Fetch filtered profiles using the service
	profiles, err := ac.MatchService.GetFilteredProfiles(r.Context(), emailId, gender, additionalFilters)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch profiles: %v", err), http.StatusInternalServerError)
		return
	}

	// Send response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"profiles": profiles,
	})
}

// GetPings handles fetching pings for a user
func (ac *MatchController) GetPings(w http.ResponseWriter, r *http.Request) {
	emailId := r.URL.Query().Get("emailId")
	if emailId == "" {
		http.Error(w, "emailId is required", http.StatusBadRequest)
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
		http.Error(w, "emailId is required", http.StatusBadRequest)
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
		http.Error(w, "emailId is required", http.StatusBadRequest)
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
