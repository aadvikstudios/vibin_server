package models

type Interaction struct {
	ReceiverHandle string `dynamodbav:"receiverHandle" json:"receiverHandle"` // ✅ Partition Key
	SenderHandle   string `dynamodbav:"senderHandle" json:"senderHandle"`     // ✅ Used in GSI
	Type           string `dynamodbav:"type" json:"type"`                     // like, ping, dislike
	Message        string `dynamodbav:"message,omitempty" json:"message,omitempty"`
	Status         string `dynamodbav:"status" json:"status"` // pending, seen
	CreatedAt      string `dynamodbav:"createdAt" json:"createdAt"`
}

// ✅ Define table name for interactions
const InteractionsTable = "Interactions"

// ✅ Define GSI Name (Used in Querying Likes/Dislikes)
const SenderHandleIndex = "senderHandle-index"
