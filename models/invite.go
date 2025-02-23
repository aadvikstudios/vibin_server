package models

const (
	InviteStatusPending  = "pending"
	InviteStatusAccepted = "accepted"
	InviteStatusDeclined = "declined"

	InviteTypeOneToOne = "one-to-one" // 1-on-1 chat
	InviteTypeGroup    = "group"      // Group chat
)

// PendingInvite represents an invite in DynamoDB
type PendingInvite struct {
	ApproverID    string `json:"approverId" dynamodbav:"approverId"`       // PK (User B approving the invite)
	CreatedAt     string `json:"createdAt" dynamodbav:"createdAt"`         // SK (Timestamp for sorting)
	InviterID     string `json:"inviterId" dynamodbav:"inviterId"`         // User A who initiated the invite
	InvitedUserID string `json:"invitedUserId" dynamodbav:"invitedUserId"` // User C being invited
	MatchID       string `json:"matchId" dynamodbav:"matchId"`             // Generated Chat ID (One-to-One or Group)
	InviteType    string `json:"inviteType" dynamodbav:"inviteType"`       // "one-to-one" or "group"
	Status        string `json:"status" dynamodbav:"status"`               // "pending", "accepted", "declined"
}

// TableName returns the DynamoDB table name
func (PendingInvite) TableName() string {
	return "PendingInvites"
}
