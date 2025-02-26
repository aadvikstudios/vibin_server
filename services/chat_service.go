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

// ✅ MarkMessagesAsRead - Marks only the messages received by user as read
func (s *ChatService) MarkMessagesAsRead(ctx context.Context, matchID string, userHandle string) error {
	log.Printf("🔄 Marking messages as read for matchId: %s where receiver is %s", matchID, userHandle)

	// ✅ Step 1: Query all messages for the given matchId
	keyCondition := "matchId = :matchId"
	expressionValues := map[string]types.AttributeValue{
		":matchId": &types.AttributeValueMemberS{Value: matchID},
	}

	// ✅ Fetch all messages
	items, err := s.Dynamo.QueryItems(ctx, models.MessagesTable, keyCondition, expressionValues, nil, 100)
	if err != nil {
		log.Printf("❌ Error fetching messages: %v", err)
		return fmt.Errorf("failed to fetch messages: %w", err)
	}

	// ✅ Step 2: Filter messages where the sender is NOT the requesting user
	var messagesToUpdate []models.Message
	for _, item := range items {
		var message models.Message
		err := attributevalue.UnmarshalMap(item, &message)
		if err != nil {
			log.Printf("⚠️ Warning: Failed to parse message: %v", err)
			continue
		}

		// ✅ Only update messages that were NOT sent by the requesting user
		if message.SenderID != userHandle && message.IsUnread == "true" {
			messagesToUpdate = append(messagesToUpdate, message)
		}
	}

	// ✅ Step 3: Batch update each message's `isUnread` status to "false"
	for _, message := range messagesToUpdate {
		// ✅ Define update key
		key := map[string]types.AttributeValue{
			"matchId":   &types.AttributeValueMemberS{Value: message.MatchID},
			"createdAt": &types.AttributeValueMemberS{Value: message.CreatedAt}, // ✅ Ensure we use the correct sort key
		}

		// ✅ Update Expression
		updateExpression := "SET isUnread = :false"
		expressionValues := map[string]types.AttributeValue{
			":false": &types.AttributeValueMemberS{Value: "false"}, // Ensure it's stored as string
		}

		// ✅ Perform update
		_, err := s.Dynamo.UpdateItem(ctx, models.MessagesTable, updateExpression, key, expressionValues, nil)
		if err != nil {
			log.Printf("❌ Failed to update message %s: %v", message.MessageID, err)
		}
	}

	log.Printf("✅ Successfully marked %d messages as read for matchId: %s where receiver is %s", len(messagesToUpdate), matchID, userHandle)
	return nil
}

// UpdateMessageLikeStatus - Updates the `liked` status of a message
func (s *ChatService) UpdateMessageLikeStatus(ctx context.Context, matchID string, createdAt string, liked bool) error {
	log.Printf("💖 Updating like status for Message at %s in MatchID: %s to %v", createdAt, matchID, liked)

	// ✅ Define the update key (Primary Key: matchId, Sort Key: createdAt)
	key := map[string]types.AttributeValue{
		"matchId":   &types.AttributeValueMemberS{Value: matchID},
		"createdAt": &types.AttributeValueMemberS{Value: createdAt}, // ✅ Correct Sort Key
	}

	// ✅ Update Expression
	updateExpression := "SET liked = :liked"
	expressionValues := map[string]types.AttributeValue{
		":liked": &types.AttributeValueMemberBOOL{Value: liked}, // ✅ Boolean type in DynamoDB
	}

	// ✅ Perform the update
	_, err := s.Dynamo.UpdateItem(ctx, models.MessagesTable, updateExpression, key, expressionValues, nil)
	if err != nil {
		log.Printf("❌ Failed to update like status: %v", err)
		return fmt.Errorf("failed to update like status: %w", err)
	}

	log.Printf("✅ Successfully updated like status for message at %s", createdAt)
	return nil
}
