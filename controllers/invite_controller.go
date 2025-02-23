package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"vibin_server/models"
	"vibin_server/services"

	"github.com/gorilla/mux"
)

// InviteController handles HTTP requests for invite-related actions
type InviteController struct {
	InviteService *services.InviteService
}

// **1️⃣ Create an Invite (Handler)**
func (c *InviteController) CreateInviteHandler(w http.ResponseWriter, r *http.Request) {
	var invite models.PendingInvite
	if err := json.NewDecoder(r.Body).Decode(&invite); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := c.InviteService.CreateInvite(context.Background(), invite.InviterID, invite.InvitedUserID, invite.ApproverID, invite.MatchID)
	if err != nil {
		http.Error(w, "Failed to create invite", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "Invite created successfully"})
}

// **2️⃣ Get Pending Invites for Approver**
func (c *InviteController) GetPendingInvitesHandler(w http.ResponseWriter, r *http.Request) {
	approverID := mux.Vars(r)["approverId"]
	invites, err := c.InviteService.GetPendingInvites(context.Background(), approverID)
	if err != nil {
		http.Error(w, "Failed to fetch invites", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(invites)
}

// **3️⃣ Get Sent Invites for Inviter**
func (c *InviteController) GetSentInvitesHandler(w http.ResponseWriter, r *http.Request) {
	inviterID := mux.Vars(r)["inviterId"]
	invites, err := c.InviteService.GetSentInvites(context.Background(), inviterID)
	if err != nil {
		http.Error(w, "Failed to fetch sent invites", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(invites)
}

// **4️⃣ Accept/Decline an Invite**
func (c *InviteController) UpdateInviteStatusHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ApproverID string `json:"approverId"`
		CreatedAt  string `json:"createdAt"`
		Status     string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := c.InviteService.UpdateInviteStatus(context.Background(), request.ApproverID, request.CreatedAt, request.Status)
	if err != nil {
		http.Error(w, "Failed to update invite", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "Invite status updated successfully"})
}
