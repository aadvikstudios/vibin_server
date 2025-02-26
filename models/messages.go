package models

type Message struct {
	MatchID   string `dynamodbav:"matchId" json:"matchId"`
	CreatedAt string `dynamodbav:"createdAt" json:"createdAt"`
	Content   string `dynamodbav:"content" json:"content"`
	IsUnread  string `dynamodbav:"isUnread" json:"isUnread"` // Store as "true" or "false"
	Liked     bool   `dynamodbav:"liked" json:"liked"`
	MessageID string `dynamodbav:"messageId" json:"messageId"`
	SenderID  string `dynamodbav:"senderId" json:"senderId"`
}

// MessagesTable is the DynamoDB table name for user messages
const MessagesTable = "Message"
