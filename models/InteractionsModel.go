package models

type Interaction struct {
	InteractionID   string   `dynamodbav:"interactionId" json:"interactionId"`         // ✅ Unique Primary Key
	Users           []string `dynamodbav:"users" json:"users"`                         // ✅ List of users involved
	UserLookup      string   `dynamodbav:"userLookup" json:"userLookup"`               // ✅ GSI-Friendly single user attribute (For GSI)
	SenderHandle    string   `dynamodbav:"senderHandle" json:"senderHandle"`           // ✅ Who initiated the interaction
	ReceiverHandle  string   `dynamodbav:"receiverHandle" json:"receiverHandle"`       // ✅ Target user (NEW FIELD)
	InteractionType string   `dynamodbav:"interactionType" json:"interactionType"`     // ✅ like, ping, invite
	ChatType        string   `dynamodbav:"chatType" json:"chatType"`                   // ✅ private, group
	Status          string   `dynamodbav:"status" json:"status"`                       // ✅ pending, match, seen
	Message         *string  `dynamodbav:"message,omitempty" json:"message,omitempty"` // ✅ Optional, only for pings or invites

	// ✅ Match-related fields
	MatchID *string `dynamodbav:"matchId,omitempty" json:"matchId,omitempty"` // ✅ Assigned when matched
	IsGroup bool    `dynamodbav:"isGroup" json:"isGroup"`                     // ✅ Differentiates group vs private chats

	// ✅ Timestamps and tracking
	CreatedAt   string  `dynamodbav:"createdAt" json:"createdAt"`                     // ✅ Timestamp of creation
	LastUpdated string  `dynamodbav:"lastUpdated" json:"lastUpdated"`                 // ✅ Updated whenever status changes
	ExpiresAt   *string `dynamodbav:"expiresAt,omitempty" json:"expiresAt,omitempty"` // ✅ TTL for auto-expiry
}

// ✅ Define table name for interactions
const InteractionsTable = "Interactions"

// ✅ Define GSI for querying interactions by user (NEW PARTITION KEY)
const UsersIndex = "users-index" // PK: userLookup

// ✅ Define GSI for querying interactions by sender
const SenderHandleIndex = "senderHandle-index" // PK: senderHandle

// ✅ Define GSI for querying interactions by match ID
const MatchIndex = "matchId-index" // PK: matchId
