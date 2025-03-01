package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
	"vibin_server/models"
	"vibin_server/services"
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

	// Create invite object
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

	// Call service layer to save invite
	if err := c.service.CreateGroupInvite(context.Background(), invite); err != nil {
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
	err := c.service.ApproveOrDeclineInvite(context.Background(), approvalRequest.ApproverHandle, approvalRequest.InviteeHandle, approvalRequest.Status)
	if err != nil {
		http.Error(w, "Failed to update invite status", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Invite status updated successfully"})
}
