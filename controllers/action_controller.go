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
func (ac *ActionController) HandleSendPing(w http.ResponseWriter, r *http.Request) {
	var request struct {
		EmailId       string `json:"emailId"`
		TargetEmailId string `json:"targetEmailId"`
		Action        string `json:"action"`
		PingNote      string `json:"pingNote"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	err := ac.ActionService.SendPing(context.Background(), request.EmailId, request.TargetEmailId, request.Action, request.PingNote)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Ping action processed successfully"})
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

// Other handler functions...
