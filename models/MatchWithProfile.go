package models

// MatchWithProfile combines Match details with the other user's profile data
type MatchWithProfile struct {
	// Match Fields
	MatchID     string `dynamodbav:"matchId" json:"matchId"`
	User1Handle string `dynamodbav:"user1Handle" json:"user1Handle"`
	User2Handle string `dynamodbav:"user2Handle" json:"user2Handle"`
	Status      string `dynamodbav:"status" json:"status"`
	CreatedAt   string `dynamodbav:"createdAt" json:"createdAt"`

	// User Profile Fields (For Matched User)
	Name            string            `json:"name,omitempty"`
	UserName        string            `json:"username,omitempty"`
	Age             int               `json:"age,omitempty"`
	Gender          string            `json:"gender,omitempty"`
	Orientation     string            `json:"orientation,omitempty"`
	LookingFor      string            `json:"lookingFor,omitempty"`
	Photos          []string          `json:"photos,omitempty"`
	Bio             string            `json:"bio,omitempty"`
	Interests       []string          `json:"interests,omitempty"`
	DistanceBetween float64           `json:"distanceBetween,omitempty"`
	Questionnaire   map[string]string `json:"questionnaire,omitempty"`
}
