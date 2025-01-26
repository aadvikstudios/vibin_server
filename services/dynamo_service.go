package services

import "time"

// Mock services to simulate DynamoDB operations

type Message struct {
	MessageID string
	MatchID   string
	Content   string
	CreatedAt time.Time
}

type UserProfile struct {
	UserID   string
	FullName string
	EmailID  string
}

func MockAddMessage() map[string]interface{} {
	return map[string]interface{}{
		"message": "Message added successfully",
		"data":    Message{MessageID: "12345", MatchID: "67890", Content: "Hello, World!", CreatedAt: time.Now()},
	}
}

func MockGetMessages(matchID string) map[string]interface{} {
	return map[string]interface{}{
		"message": "Messages fetched successfully",
		"data": []Message{
			{MessageID: "1", MatchID: matchID, Content: "Hello!", CreatedAt: time.Now()},
		},
	}
}

func MockAddUserProfile() map[string]interface{} {
	return map[string]interface{}{
		"message": "Profile added successfully",
		"data":    UserProfile{UserID: "abc123", FullName: "John Doe", EmailID: "john.doe@example.com"},
	}
}

func MockGetUserProfile(userID string) map[string]interface{} {
	return map[string]interface{}{
		"message": "Profile fetched successfully",
		"data":    UserProfile{UserID: userID, FullName: "John Doe", EmailID: "john.doe@example.com"},
	}
}

func MockDeleteUserProfile(userID string) {
	// Simulate deletion
}
