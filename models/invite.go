package models

// PendingInvite represents an invite in the PendingInvites table.
type PendingInvite struct {
	ApproverID    string `dynamodbav:"approverId"`    // PK (Email of the user who needs to approve)
	CreatedAt     string `dynamodbav:"createdAt"`     // SK (Timestamp for sorting invites)
	InviterID     string `dynamodbav:"inviterId"`     // GSI (Email of the user who initiated the invite)
	InvitedUserID string `dynamodbav:"invitedUserId"` // Email of the invited user
	MatchID       string `dynamodbav:"matchId"`       // Chat ID (one-to-one or group)
	Status        string `dynamodbav:"status"`        // Status: "pending", "accepted", "declined"
}

// TableName returns the name of the DynamoDB table
func (PendingInvite) TableName() string {
	return "PendingInvites"
}

// Possible Invite Statuses
const (
	InviteStatusPending  = "pending"
	InviteStatusAccepted = "accepted"
	InviteStatusDeclined = "declined"
)
