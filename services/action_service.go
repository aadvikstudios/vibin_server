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

// ProcessAction processes "liked" or "notliked" actions
func (as *ActionService) ProcessAction(ctx context.Context, emailId, targetEmailId, action string) (map[string]string, error) {
	switch action {
	case "liked":
		return as.handleLiked(ctx, emailId, targetEmailId)
	case "notliked":
		return as.handleNotLiked(ctx, emailId, targetEmailId)
	default:
		return nil, errors.New("invalid action")
	}
}

// AcceptPing accepts a ping and creates a match (also inserts a message)
func (as *ActionService) AcceptPing(ctx context.Context, emailId, targetEmailId, pingNote string) (map[string]string, error) {
	matchID := uuid.NewString()

	// Create match entry in DynamoDB
	err := as.createMatch(ctx, emailId, targetEmailId, matchID)
	if err != nil {
		return nil, fmt.Errorf("failed to create match: %w", err)
	}

	// Add match message in Messages table
	message := map[string]interface{}{
		"messageId": uuid.NewString(),
		"matchId":   matchID,
		"senderId":  targetEmailId,
		"content":   pingNote,
		"createdAt": time.Now().Format(time.RFC3339),
		"liked":     false,
	}

	err = as.Dynamo.PutItem(ctx, "Messages", message)
	if err != nil {
		return nil, fmt.Errorf("failed to add match message: %w", err)
	}

	return map[string]string{"message": "It's a match!", "matchId": matchID}, nil
}

// DeclinePing declines a ping request
func (as *ActionService) DeclinePing(ctx context.Context, emailId, targetEmailId string) error {
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
	return nil
}

// handleLiked processes a "liked" action
func (as *ActionService) handleLiked(ctx context.Context, emailId, targetEmailId string) (map[string]string, error) {
	// Fetch the target user's profile
	targetProfile, err := as.GetUserProfile(ctx, targetEmailId)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch target user profile: %w", err)
	}

	// Check for mutual likes to create a match
	if likedAttr, ok := targetProfile["liked"]; ok {
		likedUsers := likedAttr.(*types.AttributeValueMemberL).Value
		for _, user := range likedUsers {
			if user.(*types.AttributeValueMemberS).Value == emailId {
				// Create a match if mutual like exists
				matchID := uuid.NewString()
				err := as.createMatch(ctx, emailId, targetEmailId, matchID)
				if err != nil {
					return nil, err
				}
				return map[string]string{"message": "It's a match!", "matchId": matchID}, nil
			}
		}
	}

	// Add targetEmailId to the "liked[]" list for the emailId profile
	updateExpressionLiked := "SET liked = list_append(if_not_exists(liked, :empty), :targetEmailIdList)"
	keyLiked := map[string]types.AttributeValue{
		"emailId": &types.AttributeValueMemberS{Value: emailId},
	}
	expressionAttributeValuesLiked := map[string]types.AttributeValue{
		":empty": &types.AttributeValueMemberL{Value: []types.AttributeValue{}}, // An empty list if "liked" does not exist
		":targetEmailIdList": &types.AttributeValueMemberL{Value: []types.AttributeValue{
			&types.AttributeValueMemberS{Value: targetEmailId}, // Wrap targetEmailId in a list
		}},
	}

	// Update the liked[] list for the emailId profile
	_, err = as.Dynamo.UpdateItem(ctx, "UserProfiles", updateExpressionLiked, keyLiked, expressionAttributeValuesLiked, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to update liked list for emailId profile: %w", err)
	}

	// Add emailId to the "likedBy[]" list for the targetEmailId profile
	updateExpressionLikedBy := "SET likedBy = list_append(if_not_exists(likedBy, :empty), :emailIdList)"
	keyLikedBy := map[string]types.AttributeValue{
		"emailId": &types.AttributeValueMemberS{Value: targetEmailId},
	}
	expressionAttributeValuesLikedBy := map[string]types.AttributeValue{
		":empty": &types.AttributeValueMemberL{Value: []types.AttributeValue{}}, // An empty list if "likedBy" does not exist
		":emailIdList": &types.AttributeValueMemberL{Value: []types.AttributeValue{
			&types.AttributeValueMemberS{Value: emailId}, // Wrap emailId in a list
		}},
	}

	// Update the likedBy[] list for the targetEmailId profile
	_, err = as.Dynamo.UpdateItem(ctx, "UserProfiles", updateExpressionLikedBy, keyLikedBy, expressionAttributeValuesLikedBy, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to update likedBy list for targetEmailId profile: %w", err)
	}

	return map[string]string{"message": "User liked successfully"}, nil
}

// handleNotLiked processes a "notliked" action
func (as *ActionService) handleNotLiked(ctx context.Context, emailId, targetEmailId string) (map[string]string, error) {
	// UpdateExpression to append the targetEmailId to the "notLiked" list
	updateExpression := "SET notLiked = list_append(if_not_exists(notLiked, :empty), :targetEmailIdList)"
	key := map[string]types.AttributeValue{
		"emailId": &types.AttributeValueMemberS{Value: emailId},
	}
	expressionAttributeValues := map[string]types.AttributeValue{
		":empty": &types.AttributeValueMemberL{Value: []types.AttributeValue{}}, // An empty list if "notLiked" does not exist
		":targetEmailIdList": &types.AttributeValueMemberL{Value: []types.AttributeValue{
			&types.AttributeValueMemberS{Value: targetEmailId}, // Wrap targetEmailId in a list
		}},
	}

	// Update the item in DynamoDB
	_, err := as.Dynamo.UpdateItem(ctx, "UserProfiles", updateExpression, key, expressionAttributeValues, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to update notLiked list: %w", err)
	}

	return map[string]string{"message": "User added to notLiked list"}, nil
}

// createMatch creates a match between two users
func (as *ActionService) createMatch(ctx context.Context, emailId, targetEmailId, matchID string) error {

	// Add match entry for both users
	for _, user := range []string{emailId, targetEmailId} {
		_, err := as.Dynamo.UpdateItem(ctx, "UserProfiles", "SET matches = list_append(if_not_exists(matches, :empty), :newMatch)", map[string]types.AttributeValue{
			"emailId": &types.AttributeValueMemberS{Value: user},
		}, map[string]types.AttributeValue{
			":empty": &types.AttributeValueMemberL{Value: []types.AttributeValue{}},
			":newMatch": &types.AttributeValueMemberL{Value: []types.AttributeValue{
				&types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
					"matchId": &types.AttributeValueMemberS{Value: matchID},
					"emailId": &types.AttributeValueMemberS{Value: targetEmailId},
				}}}},
		}, nil)
		if err != nil {
			return fmt.Errorf("failed to create match: %w", err)
		}
	}

	return nil
}
