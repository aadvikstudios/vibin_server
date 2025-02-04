package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Message structure for chat messages
type Message struct {
	MessageID string `json:"messageId"`
	MatchID   string `json:"matchId"`
	SenderID  string `json:"senderId"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
	Liked     bool   `json:"liked"`
	Read      bool   `json:"read"`
}

// ChatService handles business logic for chat operations
type ChatService struct {
	Dynamo *DynamoService
}

// SaveMessage saves a new message
func (cs *ChatService) SaveMessage(message Message) error {
	if message.MatchID == "" || message.Content == "" {
		return errors.New("missing required fields")
	}
	return cs.Dynamo.PutItem(context.TODO(), "Messages", message)
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

// MarkMessagesAsRead marks all messages as read for a match ID
func (cs *ChatService) MarkMessagesAsRead(matchID string) error {
	// Fetch messages for the given matchID
	messages, err := cs.GetMessagesByMatchID(matchID)
	if err != nil {
		return fmt.Errorf("failed to fetch messages: %w", err)
	}

	for _, message := range messages {
		// Update the `isUnread` attribute to false
		updateExpression := "SET isUnread = :falseValue"
		key := map[string]types.AttributeValue{
			"matchId":   &types.AttributeValueMemberS{Value: message.MatchID},
			"messageId": &types.AttributeValueMemberS{Value: message.MessageID},
		}
		expressionAttributeValues := map[string]types.AttributeValue{
			":falseValue": &types.AttributeValueMemberBOOL{Value: false},
		}

		_, err := cs.Dynamo.UpdateItem(context.TODO(), "Messages", updateExpression, key, expressionAttributeValues, nil)
		if err != nil {
			return fmt.Errorf("failed to update isUnread for messageId %s: %w", message.MessageID, err)
		}
	}

	return nil
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
