package models

// InteractionWithProfile combines Interaction and UserProfile fields
type InteractionWithProfile struct {
	// Interaction Fields
	ReceiverHandle string `dynamodbav:"receiverHandle" json:"receiverHandle"`
	SenderHandle   string `dynamodbav:"senderHandle" json:"senderHandle"`
	Type           string `dynamodbav:"type" json:"type"` // like, ping, dislike
	Message        string `dynamodbav:"message,omitempty" json:"message,omitempty"`
	Status         string `dynamodbav:"status" json:"status"` // pending, seen
	CreatedAt      string `dynamodbav:"createdAt" json:"createdAt"`

	// User Profile Fields
	Name            string            `json:"name,omitempty"`
	UserName        string            `json:"username,omitempty"`
	Age             int               `json:"age,omitempty"`
	Gender          string            `json:"gender,omitempty"`
	Orientation     string            `json:"orientation,omitempty"`
	LookingFor      string            `json:"lookingFor,omitempty"`
	Photos          []string          `json:"photos,omitempty"`
	Bio             string            `json:"bio,omitempty"`
	Interests       []string          `json:"interests,omitempty"`
	DistanceBetween float64           `json:"distanceBetween,omitempty"` // Computed distance (not stored in DB)
	Questionnaire   map[string]string `json:"questionnaire,omitempty"`
}
