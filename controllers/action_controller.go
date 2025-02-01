package controllers

import (
	"context"
	"encoding/json"
	"log"
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

// HandlePingAction processes ping actions
func (ac *ActionController) HandlePingAction(w http.ResponseWriter, r *http.Request) {
	var request struct {
		EmailId       string `json:"emailId"`
		TargetEmailId string `json:"targetEmailId"`
		Action        string `json:"action"`
		PingNote      string `json:"pingNote"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Println("Invalid request payload:", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if request.EmailId == "" || request.TargetEmailId == "" || request.Action == "" {
		log.Println("Missing required fields in /pingAction request")
		http.Error(w, "EmailId, TargetEmailId, and Action are required", http.StatusBadRequest)
		return
	}

	response, err := ac.ActionService.ProcessPingAction(context.Background(), request.EmailId, request.TargetEmailId, request.Action, request.PingNote)
	if err != nil {
		log.Println("Error processing ping action:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// HandleAction processes user actions such as "liked", "notliked"
func (ac *ActionController) HandleAction(w http.ResponseWriter, r *http.Request) {
	var request struct {
		EmailId       string `json:"emailId"`
		TargetEmailId string `json:"targetEmailId"`
		Action        string `json:"action"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Println("Invalid request payload:", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if request.EmailId == "" || request.TargetEmailId == "" || request.Action == "" {
		log.Println("Missing required fields in /action request")
		http.Error(w, "userId, targetUserId, and action are required", http.StatusBadRequest)
		return
	}

	response, err := ac.ActionService.ProcessAction(context.Background(), request.EmailId, request.TargetEmailId, request.Action)
	if err != nil {
		log.Println("Error processing action:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
