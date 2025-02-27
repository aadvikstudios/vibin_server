// **1️⃣ Create an Invite (Handler)**
package controllers

import (
	"context"
	"encoding/json"
	"net/http"

	"vibin_server/services"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type InviteController struct {
	InviteService *services.InviteService
}

// CreateInviteHandler handles the invite creation request
func (c *InviteController) CreateInviteHandler(w http.ResponseWriter, r *http.Request) {
	var inviteRequest struct {
		InviterID     string `json:"inviterId"`     // User A (Initiator)
		InvitedUserID string `json:"invitedUserId"` // User C (New user to be added)
		ApproverID    string `json:"approverId"`    // User B (Existing chat partner who approves)
		InviteType    string `json:"inviteType"`
	}

	if err := json.NewDecoder(r.Body).Decode(&inviteRequest); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// ✅ Generate Unique Group ID (matchId)
	matchID := uuid.New().String()

	// ✅ Set status as "pending"
	err := c.InviteService.CreateInvite(
		context.Background(),
		inviteRequest.InviterID,
		inviteRequest.InvitedUserID,
		inviteRequest.ApproverID,
		inviteRequest.InviteType,
		matchID,
	)
	if err != nil {
		http.Error(w, "Failed to create invite", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Invite created successfully",
		"matchId": matchID,
	})
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
