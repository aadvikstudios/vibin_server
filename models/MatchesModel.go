package models

type Match struct {
	MatchID   string   `dynamodbav:"matchId" json:"matchId"`     // Unique matchId
	Users     []string `dynamodbav:"users" json:"users"`         // List of users (supports groups)
	Type      string   `dynamodbav:"type" json:"type"`           // "private" or "group"
	Status    string   `dynamodbav:"status" json:"status"`       // active, archived
	CreatedAt string   `dynamodbav:"createdAt" json:"createdAt"` // Timestamp of creation
}

// MatchesTable is the DynamoDB table name for user matches
const MatchesTable = "Matches"
