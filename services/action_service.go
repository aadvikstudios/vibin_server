package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

// ActionService struct
type ActionService struct {
	Dynamo *DynamoService
}

// GetUserProfile retrieves a user profile by email ID
func (as *ActionService) GetUserProfile(ctx context.Context, emailId string) (map[string]types.AttributeValue, error) {
	key := map[string]types.AttributeValue{
		"emailId": &types.AttributeValueMemberS{Value: emailId},
	}
	return as.Dynamo.GetItem(ctx, "UserProfiles", key)
}

// SendPing processes a ping action between two users
func (as *ActionService) SendPing(ctx context.Context, emailId, targetEmailId, action, pingNote string) error {
	createdAt := time.Now().UTC().Format(time.RFC3339)

	newPing := map[string]types.AttributeValue{
		"senderEmailId": &types.AttributeValueMemberS{Value: emailId},
		"pingNote":      &types.AttributeValueMemberS{Value: pingNote},
		"createdAt":     &types.AttributeValueMemberS{Value: createdAt},
	}

	updateExpression := "SET pings = list_append(if_not_exists(pings, :empty_list), :new_ping)"
	expressionAttributeValues := map[string]types.AttributeValue{
		":new_ping":   &types.AttributeValueMemberL{Value: []types.AttributeValue{&types.AttributeValueMemberM{Value: newPing}}},
		":empty_list": &types.AttributeValueMemberL{Value: []types.AttributeValue{}},
	}

	key := map[string]types.AttributeValue{
		"emailId": &types.AttributeValueMemberS{Value: targetEmailId},
	}

	_, err := as.Dynamo.UpdateItem(ctx, "UserProfiles", updateExpression, key, expressionAttributeValues, nil)
	if err != nil {
		return fmt.Errorf("failed to update target user profile with ping: %w", err)
	}

	return nil
}

// ProcessPingAction processes "accept" or "decline" ping actions
func (as *ActionService) ProcessPingAction(ctx context.Context, emailId, targetEmailId, action, pingNote string) (map[string]string, error) {
	switch action {
	case "accept":
		return as.AcceptPing(ctx, emailId, targetEmailId, pingNote)
	case "decline":
		err := as.DeclinePing(ctx, emailId, targetEmailId)
		if err != nil {
			return nil, err
		}
		return map[string]string{"message": "Ping declined"}, nil
	default:
		return nil, errors.New("invalid action")
	}
}

// AcceptPing handles the acceptance of a ping
func (as *ActionService) AcceptPing(ctx context.Context, emailId, targetEmailId, pingNote string) (map[string]string, error) {
	matchID := uuid.NewString()

	// Create match entry in DynamoDB
	err := as.createMatch(ctx, emailId, targetEmailId, matchID)
	if err != nil {
		return nil, fmt.Errorf("failed to create match: %w", err)
	}

	// Use CreateMessage to insert a message
	err = as.CreateMessage(ctx, matchID, targetEmailId, pingNote, false, true)
	if err != nil {
		return nil, fmt.Errorf("failed to add match message: %w", err)
	}

	// Remove the ping after acceptance
	err = as.removeFromList(ctx, emailId, "pings", targetEmailId)
	if err != nil {
		return nil, fmt.Errorf("failed to remove ping after acceptance: %w", err)
	}

	return map[string]string{"message": "It's a match!", "matchId": matchID}, nil
}

// DeclinePing declines a ping request
func (as *ActionService) DeclinePing(ctx context.Context, emailId, targetEmailId string) error {
	// Append the targetEmailId to the "notLiked" list
	updateExpression := "SET notLiked = list_append(if_not_exists(notLiked, :empty), :targetEmailId)"
	_, err := as.Dynamo.UpdateItem(ctx, "UserProfiles", updateExpression, map[string]types.AttributeValue{
		"emailId": &types.AttributeValueMemberS{Value: emailId},
	}, map[string]types.AttributeValue{
		":empty":         &types.AttributeValueMemberL{Value: []types.AttributeValue{}},
		":targetEmailId": &types.AttributeValueMemberS{Value: targetEmailId},
	}, nil)

	if err != nil {
		return fmt.Errorf("failed to decline ping: %w", err)
	}

	// Remove the ping after declining
	err = as.removeFromList(ctx, emailId, "pings", targetEmailId)
	if err != nil {
		return fmt.Errorf("failed to remove ping after decline: %w", err)
	}

	return nil
}

// CreateMatch creates a match entry for two users
func (as *ActionService) createMatch(ctx context.Context, emailId, targetEmailId, matchID string) error {
	matchData := map[string]map[string]types.AttributeValue{
		emailId: {
			"matchId": &types.AttributeValueMemberS{Value: matchID},
			"emailId": &types.AttributeValueMemberS{Value: targetEmailId},
		},
		targetEmailId: {
			"matchId": &types.AttributeValueMemberS{Value: matchID},
			"emailId": &types.AttributeValueMemberS{Value: emailId},
		},
	}

	for user, match := range matchData {
		_, err := as.Dynamo.UpdateItem(ctx, "UserProfiles",
			"SET matches = list_append(if_not_exists(matches, :empty), :newMatch)",
			map[string]types.AttributeValue{
				"emailId": &types.AttributeValueMemberS{Value: user},
			},
			map[string]types.AttributeValue{
				":empty": &types.AttributeValueMemberL{Value: []types.AttributeValue{}},
				":newMatch": &types.AttributeValueMemberL{Value: []types.AttributeValue{
					&types.AttributeValueMemberM{Value: match},
				}},
			}, nil)

		if err != nil {
			return fmt.Errorf("failed to create match for user %s: %w", user, err)
		}
	}

	return nil
}

// CreateMessage adds a new message to the Messages table
func (as *ActionService) CreateMessage(ctx context.Context, matchID, senderID, content string, liked bool, isUnread bool) error {
	message := map[string]interface{}{
		"messageId": uuid.NewString(),
		"matchId":   matchID,
		"senderId":  senderID,
		"content":   content,
		"createdAt": time.Now().Format(time.RFC3339),
		"liked":     liked,
		"isUnread":  isUnread,
	}

	err := as.Dynamo.PutItem(ctx, "Messages", message)
	if err != nil {
		return fmt.Errorf("failed to add message: %w", err)
	}
	return nil
}

// Reusable method to remove an item from a list
func (as *ActionService) removeFromList(ctx context.Context, emailId, listName, targetValue string) error {
	profile, err := as.GetUserProfile(ctx, emailId)
	if err != nil {
		return fmt.Errorf("failed to fetch user profile: %w", err)
	}

	listAttr, exists := profile[listName]
	if !exists {
		return fmt.Errorf("list '%s' not found", listName)
	}

	listValues, ok := listAttr.(*types.AttributeValueMemberL)
	if !ok || len(listValues.Value) == 0 {
		return fmt.Errorf("list '%s' is empty", listName)
	}

	var itemIndex int = -1
	for i, item := range listValues.Value {
		if item.(*types.AttributeValueMemberS).Value == targetValue {
			itemIndex = i
			break
		}
	}

	if itemIndex == -1 {
		return fmt.Errorf("target value not found in list '%s'", listName)
	}

	updateExpression := fmt.Sprintf("REMOVE %s[%d]", listName, itemIndex)

	_, err = as.Dynamo.UpdateItem(ctx, "UserProfiles", updateExpression, map[string]types.AttributeValue{
		"emailId": &types.AttributeValueMemberS{Value: emailId},
	}, nil, nil)

	if err != nil {
		return fmt.Errorf("failed to remove item from list: %w", err)
	}

	return nil
}
