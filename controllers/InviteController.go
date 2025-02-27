package controllers

import (
	"context"
	"encoding/json"
	"net/http"

	"vibin_server/models"
	"vibin_server/services"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type InviteController struct {
	InviteService *services.InviteService
}

// **Create an Invite (User A invites User C & requests approval from User B)**
func (c *InviteController) CreateInviteHandler(w http.ResponseWriter, r *http.Request) {
	var inviteRequest struct {
		InviterID     string `json:"inviterId"`     // User A (Initiator)
		InvitedUserID string `json:"invitedUserId"` // User C (New user to be added)
		ApproverID    string `json:"approverId"`    // User B (Existing chat partner who approves)
		InviteType    string `json:"inviteType"`    // "group"
	}

	if err := json.NewDecoder(r.Body).Decode(&inviteRequest); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// ✅ Generate a new MatchId for the new group chat
	newMatchID := uuid.New().String()

	// ✅ Store the invite with the new matchId
	err := c.InviteService.CreateInvite(
		context.Background(),
		inviteRequest.InviterID,
		inviteRequest.InvitedUserID,
		inviteRequest.ApproverID,
		inviteRequest.InviteType,
		newMatchID, // 🔹 New matchId for the group chat
	)
	if err != nil {
		http.Error(w, "Failed to create invite", http.StatusInternalServerError)
		return
	}

	// ✅ Return the new matchId
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Invite created successfully",
		"matchId": newMatchID,
	})
}

// **Handle Invite Approval or Decline (User B's Action)**
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

	// ✅ Fetch the invite details
	invite, err := c.InviteService.GetInviteByApproverAndTime(context.Background(), request.ApproverID, request.CreatedAt)
	if err != nil {
		http.Error(w, "Invite not found", http.StatusNotFound)
		return
	}

	// ✅ If the invite is accepted, create a new group chat matchId
	if request.Status == models.InviteStatusAccepted {
		newMatchID := invite.MatchID // Already generated at the time of invite creation

		// ✅ Create the new group match
		err = c.InviteService.CreateGroupMatch(context.Background(), newMatchID, []string{invite.InviterID, invite.ApproverID, invite.InvitedUserID})
		if err != nil {
			http.Error(w, "Failed to create group chat", http.StatusInternalServerError)
			return
		}
	}

	// ✅ Update the invite status
	err = c.InviteService.UpdateInviteStatus(context.Background(), request.ApproverID, request.CreatedAt, request.Status)
	if err != nil {
		http.Error(w, "Failed to update invite", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "Invite status updated successfully"})
}

// **Get Pending Invites for Approver**
func (c *InviteController) GetPendingInvitesHandler(w http.ResponseWriter, r *http.Request) {
	approverID := mux.Vars(r)["approverId"]
	invites, err := c.InviteService.GetPendingInvites(context.Background(), approverID)
	if err != nil {
		http.Error(w, "Failed to fetch invites", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(invites)
}

// **Get Sent Invites for Inviter**
func (c *InviteController) GetSentInvitesHandler(w http.ResponseWriter, r *http.Request) {
	inviterID := mux.Vars(r)["inviterId"]
	invites, err := c.InviteService.GetSentInvites(context.Background(), inviterID)
	if err != nil {
		http.Error(w, "Failed to fetch sent invites", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(invites)
}
