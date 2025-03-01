package models

import "time"

// GroupInteraction represents a group invite or an approved group in DynamoDB
type GroupInteraction struct {
	PK              string              `dynamodbav:"PK" json:"PK"`                               // "USER#<userHandle>"
	SK              string              `dynamodbav:"SK" json:"SK"`                               // "GROUP_INVITE#<inviteeHandle>" OR "GROUP#<groupId>"
	InteractionType string              `dynamodbav:"interactionType" json:"interactionType"`     // "group_invite" or "group_chat"
	Status          string              `dynamodbav:"status" json:"status"`                       // "pending", "approved", "active"
	GroupID         *string             `dynamodbav:"groupId,omitempty" json:"groupId,omitempty"` // Assigned when approved
	InviterHandle   string              `dynamodbav:"inviterHandle" json:"inviterHandle"`         // User who initiated the invite
	ApproverHandle  string              `dynamodbav:"approverHandle" json:"approverHandle"`       // User who needs to approve
	InviteeHandle   string              `dynamodbav:"inviteeHandle" json:"inviteeHandle"`         // User who will be added
	Members         []string            `dynamodbav:"members" json:"members"`                     // List of users in the group
	CreatedAt       time.Time           `dynamodbav:"createdAt" json:"createdAt"`                 // Timestamp of invite creation
	LastUpdated     time.Time           `dynamodbav:"lastUpdated" json:"lastUpdated"`             // Timestamp of last update
	InviteeProfile  *InviteeUserDetails `json:"inviteeProfile,omitempty"`                         // Invitee's profile details
}

// MatchedUserDetails represents the necessary data for a matched user
type InviteeUserDetails struct {
	Name        string   `json:"name"`
	Photo       string   `json:"photo"`
	Bio         string   `json:"bio,omitempty"`
	Desires     []string `json:"desires,omitempty"`
	Gender      string   `json:"gender,omitempty"`
	Interests   []string `json:"interests,omitempty"`
	LookingFor  string   `json:"lookingFor,omitempty"`
	Orientation string   `json:"orientation,omitempty"`
}

// Table Name for DynamoDB
const GroupInteractionsTable = "GroupInteractions"

// GSI Index Names
const InviteStatusIndex = "inviterHandle-status-index" // GSI for querying invite status
const ApprovalIndex = "approverHandle-status-index"    // GSI for querying pending approvals
