package models

import "strings"

// Message represents a chat message stored in DynamoDB
type Message struct {
	MatchID   string `dynamodbav:"matchId" json:"matchId"`
	CreatedAt string `dynamodbav:"createdAt" json:"createdAt"`
	Content   string `dynamodbav:"content" json:"content"`
	IsUnread  string `dynamodbav:"isUnread" json:"isUnread"` // ✅ Stored as "true" or "false"
	Liked     bool   `dynamodbav:"liked" json:"liked"`
	MessageID string `dynamodbav:"messageId" json:"messageId"`
	SenderID  string `dynamodbav:"senderId" json:"senderId"`
	ImageURL  string `dynamodbav:"imageUrl,omitempty" json:"imageUrl,omitempty"` // ✅ New Field for Image Messages
}

// MessagesTable is the DynamoDB table name
const MessagesTable = "Message"

// ✅ Convert `isUnread` to boolean in Go
func (m *Message) IsUnreadBool() bool {
	return strings.ToLower(m.IsUnread) == "true"
}

// ✅ Convert boolean back to string before saving to DB
func (m *Message) SetIsUnread(value bool) {
	if value {
		m.IsUnread = "true"
	} else {
		m.IsUnread = "false"
	}
}
