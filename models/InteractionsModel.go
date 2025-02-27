package models

type Interaction struct {
	PK              string  `dynamodbav:"PK" json:"PK"`                               // ✅ Partition Key: "USER#sender"
	SK              string  `dynamodbav:"SK" json:"SK"`                               // ✅ Sort Key: "INTERACTION#receiver"
	SenderHandle    string  `dynamodbav:"senderHandle" json:"senderHandle"`           // ✅ Who initiated the interaction
	ReceiverHandle  string  `dynamodbav:"receiverHandle" json:"receiverHandle"`       // ✅ Target user
	InteractionType string  `dynamodbav:"interactionType" json:"interactionType"`     // ✅ like, ping, invite
	Status          string  `dynamodbav:"status" json:"status"`                       // ✅ pending, match, seen
	MatchID         *string `dynamodbav:"matchId,omitempty" json:"matchId,omitempty"` // ✅ Assigned when matched
	Message         *string `dynamodbav:"message,omitempty" json:"message,omitempty"` // ✅ Optional, only for pings or invites
	CreatedAt       string  `dynamodbav:"createdAt" json:"createdAt"`                 // ✅ Timestamp of creation
	LastUpdated     string  `dynamodbav:"lastUpdated" json:"lastUpdated"`             // ✅ Updated when status changes
}

// ✅ Define table name
const InteractionsTable = "Interactions"

// ✅ Define GSI for querying interactions where the user is the receiver
const ReceiverHandleIndex = "receiverHandle-index" // PK: receiverHandle
