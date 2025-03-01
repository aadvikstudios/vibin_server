package controllers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
	"vibin_server/models"
	"vibin_server/services"

	"github.com/gorilla/mux"
)

// GroupInteractionController handles group invite operations
type GroupInteractionController struct {
	service *services.GroupInteractionService
}

// NewGroupInteractionController creates a new instance of the controller
func NewGroupInteractionController(service *services.GroupInteractionService) *GroupInteractionController {
	return &GroupInteractionController{service: service}
}

// ✅ CreateGroupInvite - Handles the creation of a group invite
func (c *GroupInteractionController) CreateGroupInvite(w http.ResponseWriter, r *http.Request) {
	var inviteRequest struct {
		InviterHandle  string `json:"inviterHandle"`
		ApproverHandle string `json:"approverHandle"`
		InviteeHandle  string `json:"inviteeHandle"`
	}

	// Decode request body
	if err := json.NewDecoder(r.Body).Decode(&inviteRequest); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// ✅ Validate that all required fields are provided
	if inviteRequest.InviterHandle == "" || inviteRequest.ApproverHandle == "" || inviteRequest.InviteeHandle == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// ✅ Call service layer to validate and save invite
	invite := models.GroupInteraction{
		PK:              "USER#" + inviteRequest.InviterHandle,
		SK:              "GROUP_INVITE#" + inviteRequest.InviteeHandle,
		InteractionType: "group_invite",
		Status:          "pending",
		GroupID:         nil, // No group ID yet
		InviterHandle:   inviteRequest.InviterHandle,
		ApproverHandle:  inviteRequest.ApproverHandle,
		InviteeHandle:   inviteRequest.InviteeHandle,
		Members:         []string{inviteRequest.InviterHandle, inviteRequest.ApproverHandle},
		CreatedAt:       time.Now(),
		LastUpdated:     time.Now(),
	}

	err := c.service.CreateGroupInvite(context.Background(), invite)
	if err != nil {
		// ✅ Handle invalid invitee case separately
		if err.Error() == "invalid_invitee_handle" {
			http.Error(w, "Invitee handle does not exist", http.StatusNotFound)
			return
		}

		// ✅ Return generic internal error if anything else fails
		http.Error(w, "Failed to create group invite", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Group invite created successfully"})
}

// ✅ GetSentInvites - Fetches invites created by User A
func (c *GroupInteractionController) GetSentInvites(w http.ResponseWriter, r *http.Request) {
	// Extract user from query params
	userHandle := r.URL.Query().Get("userHandle")
	if userHandle == "" {
		http.Error(w, "userHandle is required", http.StatusBadRequest)
		return
	}

	// Fetch invites from service layer
	invites, err := c.service.GetSentInvites(context.Background(), userHandle)
	if err != nil {
		http.Error(w, "Failed to fetch invites", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(invites)
}

// ✅ GetPendingApprovals - Fetches invites pending approval for User B
func (c *GroupInteractionController) GetPendingApprovals(w http.ResponseWriter, r *http.Request) {
	// Extract approver from query params
	approverHandle := r.URL.Query().Get("approverHandle")
	if approverHandle == "" {
		http.Error(w, "approverHandle is required", http.StatusBadRequest)
		return
	}

	// Fetch pending approvals from service layer
	pendingInvites, err := c.service.GetPendingApprovals(context.Background(), approverHandle)
	if err != nil {
		http.Error(w, "Failed to fetch pending approvals", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(pendingInvites)
}

// ✅ ApproveOrDeclineInvite - Handles approving or declining an invite
func (c *GroupInteractionController) ApproveOrDeclineInvite(w http.ResponseWriter, r *http.Request) {
	var approvalRequest struct {
		ApproverHandle string `json:"approverHandle"`
		InviterHandle  string `json:"inviterHandle"`
		InviteeHandle  string `json:"inviteeHandle"`
		Status         string `json:"status"` // "approved" or "declined"
	}

	// Decode request body
	if err := json.NewDecoder(r.Body).Decode(&approvalRequest); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Validate status
	if approvalRequest.Status != "approved" && approvalRequest.Status != "declined" {
		http.Error(w, "Invalid status value", http.StatusBadRequest)
		return
	}

	// Call service layer to approve/decline invite
	err := c.service.ApproveOrDeclineInvite(context.Background(), approvalRequest.ApproverHandle, approvalRequest.InviterHandle, approvalRequest.InviteeHandle, approvalRequest.Status)
	if err != nil {
		http.Error(w, "Failed to update invite status", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Invite status updated successfully"})
}

// ✅ GetActiveGroups - Fetches all active groups for a given user
func (c *GroupInteractionController) GetActiveGroups(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userHandle := vars["userHandle"]

	log.Printf("🔍 Fetching active groups for user: %s", userHandle)

	groups, err := c.service.GetActiveGroups(r.Context(), userHandle)
	if err != nil {
		log.Printf("❌ Error fetching active groups for %s: %v", userHandle, err)
		http.Error(w, "Failed to fetch active groups", http.StatusInternalServerError)
		return
	}

	// ✅ Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(groups)
}
