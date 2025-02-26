package services

import (
	"context"
	"fmt"
	"log"
	"vibin_server/models"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// ChatService struct
type ChatService struct {
	Dynamo *DynamoService
}

// GetMessagesByMatchID fetches the latest messages for a given matchId sorted by createdAt (latest first),
// then reverses the order before returning, so the latest message appears at the bottom in UI.
func (s *ChatService) GetMessagesByMatchID(ctx context.Context, matchID string, limit int) ([]models.Message, error) {
	log.Printf("🔍 Fetching latest %d messages for matchId: %s", limit, matchID)

	// ✅ Define key condition expression for filtering by matchId
	keyCondition := "#matchId = :matchId"
	expressionValues := map[string]types.AttributeValue{
		":matchId": &types.AttributeValueMemberS{Value: matchID},
	}
	expressionNames := map[string]string{
		"#matchId": "matchId", // ✅ Prevents DynamoDB reserved word conflicts
	}

	// ✅ Query DynamoDB (Retrieve latest messages first)
	items, err := s.Dynamo.QueryItemsWithOptions(ctx, models.MessagesTable, keyCondition, expressionValues, expressionNames, int32(limit), true)
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

	// ✅ Reverse the messages so latest appears at the bottom in UI
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	log.Printf("✅ Found %d messages for matchId: %s, returning in UI-friendly order", len(messages), matchID)
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
