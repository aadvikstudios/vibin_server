package services

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"vibin_server/models"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// ChatService struct
type ChatService struct {
	Dynamo *DynamoService
}

// GetMessagesByMatchID fetches messages for a given matchId sorted by createdAt
func (s *ChatService) GetMessagesByMatchID(ctx context.Context, matchID string, limit int) ([]models.Message, error) {
	log.Printf("🔍 Fetching messages for matchId: %s, Limit: %d", matchID, limit)

	// ✅ Define the key condition expression
	keyCondition := "#matchId = :matchId"
	expressionValues := map[string]types.AttributeValue{
		":matchId": &types.AttributeValueMemberS{Value: matchID},
	}
	expressionNames := map[string]string{
		"#matchId": "matchId", // ✅ Prevents DynamoDB reserved word conflicts
	}

	// ✅ Convert `limit` from `int` to `int32`
	limitInt32 := int32(limit)

	// ✅ Query DynamoDB (Fixed argument count)
	items, err := s.Dynamo.QueryItems(ctx, models.MessagesTable, keyCondition, expressionValues, expressionNames, limitInt32)
	if err != nil {
		log.Printf("❌ Error querying messages: %v", err)
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	// ✅ Unmarshal results
	var messages []models.Message
	err = attributevalue.UnmarshalListOfMaps(items, &messages)
	if err != nil {
		log.Printf("❌ Error unmarshalling messages: %v", err)
		return nil, fmt.Errorf("failed to parse messages: %w", err)
	}

	// ✅ Sort results manually (since DynamoDB doesn't provide order directly)
	// Sorting in descending order (newest first)
	sort.SliceStable(messages, func(i, j int) bool {
		return messages[i].CreatedAt > messages[j].CreatedAt
	})

	// ✅ Convert `isUnread` to lowercase for consistency
	for i, msg := range messages {
		messages[i].IsUnread = strings.ToLower(msg.IsUnread) // Ensure "True" -> "true"
	}

	log.Printf("✅ Found %d messages for matchId: %s", len(messages), matchID)
	return messages, nil
}

// SendMessage stores a new message in the Messages table
func (s *ChatService) SendMessage(ctx context.Context, message models.Message) error {
	// ✅ Ensure `isUnread` is stored as a string
	message.SetIsUnread(true) // Default new messages to unread

	log.Printf("📩 Storing message: %+v", message)

	// ✅ Save message to DynamoDB
	err := s.Dynamo.PutItem(ctx, models.MessagesTable, message)
	if err != nil {
		log.Printf("❌ Failed to store message: %v", err)
		return fmt.Errorf("failed to store message: %w", err)
	}

	log.Printf("✅ Message stored successfully")
	return nil
}
