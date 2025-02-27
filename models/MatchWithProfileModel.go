package models

// MatchWithProfile combines Match details with participants' profile data
type MatchWithProfile struct {
	// Match Fields
	MatchID   string   `dynamodbav:"matchId" json:"matchId"`     // Unique matchId
	Users     []string `dynamodbav:"users" json:"users"`         // List of users (supports groups)
	Type      string   `dynamodbav:"type" json:"type"`           // "private" or "group"
	Status    string   `dynamodbav:"status" json:"status"`       // active, archived
	CreatedAt string   `dynamodbav:"createdAt" json:"createdAt"` // Timestamp of creation

	// User Profile Fields (For Participants)
	UserProfiles []UserProfile `json:"userProfiles,omitempty"` // ✅ Uses existing UserProfile struct

	// New Fields
	LastMessage string `json:"lastMessage,omitempty"` // ✅ Last message content
	IsUnread    bool   `json:"isUnread,omitempty"`    // ✅ If user has unread messages
}
