package services

import (
	"context"
	"fmt"
	"log"
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
	log.Printf("🔍 Querying messages for matchId: %s (Limit: %d)", matchID, limit)

	keyCondition := "matchId = :matchId"
	expressionValues := map[string]types.AttributeValue{
		":matchId": &types.AttributeValueMemberS{Value: matchID},
	}

	// ✅ Convert `limit` from `int` to `int32`
	limitInt32 := int32(limit)

	// ✅ Query DynamoDB
	items, err := s.Dynamo.QueryItems(ctx, models.MessagesTable, keyCondition, expressionValues, map[string]string{"createdAt": "DESC"}, limitInt32)
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

	// ✅ Convert `isUnread` from string to lowercase for consistency
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
