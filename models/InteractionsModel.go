package models

type Interaction struct {
	ReceiverHandle string  `dynamodbav:"receiverHandle" json:"receiverHandle"` // ✅ Partition Key (PK)
	SortKey        string  `dynamodbav:"sk" json:"sk"`                         // ✅ Sort Key (senderHandle#type)
	SenderHandle   string  `dynamodbav:"senderHandle" json:"senderHandle"`
	Type           string  `dynamodbav:"type" json:"type"`                           // like, ping, dislike
	Message        *string `dynamodbav:"message,omitempty" json:"message,omitempty"` // Optional for pings
	Status         string  `dynamodbav:"status" json:"status"`                       // pending, seen
	CreatedAt      string  `dynamodbav:"createdAt" json:"createdAt"`
}

// ✅ Define table name for interactions
const InteractionsTable = "Interactions"

// ✅ Define GSI for querying by senderHandle
const SenderHandleIndex = "senderHandle-index"
