package models

// GroupMessage represents a group chat message stored in DynamoDB
type GroupMessage struct {
	GroupID     string          `dynamodbav:"groupId" json:"groupId"`     // ✅ Partition Key (Group Identifier)
	CreatedAt   string          `dynamodbav:"createdAt" json:"createdAt"` // ✅ Sort Key (Timestamp)
	MessageID   string          `dynamodbav:"messageId" json:"messageId"` // ✅ Unique message ID (UUID-based)
	SenderID    string          `dynamodbav:"senderId" json:"senderId"`   // ✅ User who sent the message
	Content     string          `dynamodbav:"content,omitempty" json:"content,omitempty"`
	ImageURL    *string         `dynamodbav:"imageUrl,omitempty" json:"imageUrl,omitempty"` // ✅ Optional Image URL
	IsRead      map[string]bool `dynamodbav:"isRead" json:"isRead"`                         // ✅ Tracks read status per user
	Likes       map[string]bool `dynamodbav:"likes" json:"likes"`                           // ✅ Tracks likes per user
	ReadCount   int             `dynamodbav:"readCount" json:"readCount"`                   // ✅ Number of users who have read the message
	LikeCount   int             `dynamodbav:"likeCount" json:"likeCount"`                   // ✅ Number of users who liked the message
	MemberCount int             `dynamodbav:"memberCount" json:"memberCount"`               // ✅ Total members in the group
}

// Table Name for DynamoDB
const GroupMessageTable = "GroupMessages"
