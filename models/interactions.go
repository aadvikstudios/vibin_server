package models

type Interaction struct {
	InteractionID  string `dynamodbav:"interactionId" json:"interactionId"`
	ReceiverHandle string `dynamodbav:"receiverHandle" json:"receiverHandle"`
	SenderHandle   string `dynamodbav:"senderHandle" json:"senderHandle"`
	Type           string `dynamodbav:"type" json:"type"` // like, ping, dislike
	Message        string `dynamodbav:"message,omitempty" json:"message,omitempty"`
	Status         string `dynamodbav:"status" json:"status"` // pending, seen
	CreatedAt      string `dynamodbav:"createdAt" json:"createdAt"`
}

// InteractionsTable is the DynamoDB table name for user interactions
const InteractionsTable = "Interactions"
