package models

type Match struct {
	MatchID     string `dynamodbav:"matchId" json:"matchId"`
	User1Handle string `dynamodbav:"user1Handle" json:"user1Handle"`
	User2Handle string `dynamodbav:"user2Handle" json:"user2Handle"`
	Status      string `dynamodbav:"status" json:"status"` // active, archived
	CreatedAt   string `dynamodbav:"createdAt" json:"createdAt"`
}

// MatchesTable is the DynamoDB table name for user matches
const MatchesTable = "Matches"
