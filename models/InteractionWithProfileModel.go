package models

// InteractionWithProfile combines minimal Interaction and UserProfile fields
type InteractionWithProfile struct {
	// Interaction Fields
	ReceiverHandle  string `json:"receiverHandle"`
	SenderHandle    string `json:"senderHandle"`
	InteractionType string `dynamodbav:"interactionType" json:"interactionType"` // like, ping, dislike
	Message         string `json:"message,omitempty"`
	Status          string `json:"status"`
	CreatedAt       string `json:"createdAt"`

	// Extracted User Profile Fields
	Name            string   `json:"name,omitempty"`
	Age             int      `json:"age,omitempty"`
	Gender          string   `json:"gender,omitempty"`
	Orientation     string   `json:"orientation,omitempty"`
	LookingFor      string   `json:"lookingFor,omitempty"`
	Photos          []string `json:"photos,omitempty"`
	Bio             string   `json:"bio,omitempty"`
	Interests       []string `json:"interests,omitempty"`
	DistanceBetween float64  `json:"distanceBetween,omitempty"` // Computed distance (not stored in DB)
}

// MatchedUserDetails represents the necessary data for a matched user
type MatchedUserDetails struct {
	Name       string `json:"name"`
	UserHandle string `json:"userHandle"`
	Photo      string `json:"photo"`
	MatchID    string `json:"matchId"`
}
