package models

// PendingInvite represents an invite for a third user to join an existing chat
type PendingInvite struct {
	MatchID       string `dynamodbav:"matchId" json:"matchId"`             // Partition Key (PK) - Unique match ID
	CreatedAt     string `dynamodbav:"createdAt" json:"createdAt"`         // Sort Key (SK) - Invite creation timestamp
	InviterID     string `dynamodbav:"inviterId" json:"inviterId"`         // User who initiated the invite
	ApproverID    string `dynamodbav:"approverId" json:"approverId"`       // User who needs to approve
	InvitedUserID string `dynamodbav:"invitedUserId" json:"invitedUserId"` // The user being invited
	InviteType    string `dynamodbav:"inviteType" json:"inviteType"`       // "group" (group chat invite)
	Status        string `dynamodbav:"status" json:"status"`               // "pending", "accepted", "declined"
}

// Invite Status Constants
const (
	InviteStatusPending  = "pending"
	InviteStatusAccepted = "accepted"
	InviteStatusDeclined = "declined"
)

// TableName returns the DynamoDB table name for the PendingInvite model
func (PendingInvite) TableName() string {
	return "PendingInvites" // Ensure this matches the table name in DynamoDB
}
