package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Message structure for chat messages
type Message struct {
	MessageID string `json:"messageId"`
	MatchID   string `json:"matchId"`
	SenderID  string `json:"senderId"`
	Content   string `json:"content"`
	ImageURL  string `json:"imageUrl"`
	CreatedAt string `json:"createdAt"`
	Liked     bool   `json:"liked"`
	Read      bool   `json:"isUnRead"`
}

// ChatService handles business logic for chat operations
type ChatService struct {
	Dynamo *DynamoService
}

// SaveMessage saves a new message in the database
func (cs *ChatService) SaveMessage(message Message) error {
	// Ensure matchId and senderId are provided
	if message.MatchID == "" || message.SenderID == "" {
		return errors.New("missing required fields: matchId or senderId")
	}

	// Ensure at least one of content or imageUrl is provided
	if message.Content == "" && message.ImageURL == "" {
		return errors.New("either content or imageUrl must be provided")
	}

	// Save message to DynamoDB
	return cs.Dynamo.PutItem(context.TODO(), "Messages", message)
}

// MarkMessagesAsRead marks all messages as read for a match ID// MarkMessagesAsRead marks all messages as read for a match ID
func (cs *ChatService) MarkMessagesAsRead(matchID string) error {
	// Fetch messages for the given matchID
	messages, err := cs.GetMessagesByMatchID(matchID)
	if err != nil {
		return fmt.Errorf("failed to fetch messages: %w", err)
	}

	// Debug: Log the fetched messages
	fmt.Printf("[DEBUG] MarkMessagesAsRead: Fetched messages for matchID %s: %+v\n", matchID, messages)

	// Iterate over each message and update only the "isUnread" field
	for _, message := range messages {
		// Debug: Log the message being updated
		fmt.Printf("[DEBUG] MarkMessagesAsRead: Updating messageId %s\n", message.MessageID)

		// Prepare the update expression
		updateExpression := "SET isUnread = :falseValue"
		key := map[string]types.AttributeValue{
			"matchId":   &types.AttributeValueMemberS{Value: message.MatchID},   // Partition key
			"createdAt": &types.AttributeValueMemberS{Value: message.CreatedAt}, // Sort key
		}
		expressionAttributeValues := map[string]types.AttributeValue{
			":falseValue": &types.AttributeValueMemberBOOL{Value: false},
		}

		// Perform the update
		_, err := cs.Dynamo.Client.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
			TableName:                 aws.String("Messages"),
			Key:                       key,
			UpdateExpression:          aws.String(updateExpression),
			ExpressionAttributeValues: expressionAttributeValues,
		})
		if err != nil {
			fmt.Printf("[ERROR] MarkMessagesAsRead: Failed to update messageId %s: %v\n", message.MessageID, err)
			return fmt.Errorf("failed to update messageId %s: %w", message.MessageID, err)
		}
	}

	// Debug: Log success
	fmt.Printf("[DEBUG] MarkMessagesAsRead: Successfully marked messages as read for matchID %s\n", matchID)
	return nil
}

// GetMessagesByMatchID fetches messages by match ID
func (cs *ChatService) GetMessagesByMatchID(matchID string) ([]Message, error) {
	items, err := cs.Dynamo.QueryItems(context.TODO(), "Messages", "matchId = :matchId", map[string]types.AttributeValue{
		":matchId": &types.AttributeValueMemberS{Value: matchID},
	}, nil, 20)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}

	var messages []Message
	for _, item := range items {
		var message Message
		if err := attributevalue.UnmarshalMap(item, &message); err != nil {
			return nil, fmt.Errorf("failed to unmarshal message: %w", err)
		}
		messages = append(messages, message)
	}
	return messages, nil
}

// LikeMessage likes or unlikes a message
func (cs *ChatService) LikeMessage(matchID, messageID, createdAt string, liked bool) error {
	_, err := cs.Dynamo.UpdateItem(context.TODO(), "Messages", "SET liked = :liked", map[string]types.AttributeValue{
		"matchId":   &types.AttributeValueMemberS{Value: matchID},
		"createdAt": &types.AttributeValueMemberS{Value: createdAt},
	}, map[string]types.AttributeValue{
		":liked": &types.AttributeValueMemberBOOL{Value: liked},
	}, nil)
	return err
}
