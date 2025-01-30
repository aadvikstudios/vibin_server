package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

type ActionService struct {
	Dynamo *DynamoService
}

// GetUserProfile retrieves a user profile by ID
func (as *ActionService) GetUserProfile(ctx context.Context, userID string) (map[string]types.AttributeValue, error) {
	key := map[string]types.AttributeValue{
		"userId": &types.AttributeValueMemberS{Value: userID},
	}
	return as.Dynamo.GetItem(ctx, "UserProfiles", key)
}

// PingAction processes a ping action between two users
func (as *ActionService) PingAction(ctx context.Context, emailId string, targetEmailId string, action string, pingNote string) error {
	// Retrieve sender's profile
	senderProfile, err := as.GetUserProfile(ctx, emailId)
	if err != nil || senderProfile == nil {
		return errors.New("sender profile not found")
	}

	// Extract sender details
	senderName := extractString(senderProfile, "name")
	senderGender := extractString(senderProfile, "gender")
	senderPhoto := extractFirstPhoto(senderProfile, "photos")

	// Construct ping object
	newPing := map[string]types.AttributeValue{
		"senderEmailId": &types.AttributeValueMemberS{Value: emailId},
		"name":          &types.AttributeValueMemberS{Value: senderName},
		"gender":        &types.AttributeValueMemberS{Value: senderGender},
		"photo":         &types.AttributeValueMemberS{Value: senderPhoto},
		"pingNote":      &types.AttributeValueMemberS{Value: pingNote},
	}

	// Append new ping to target user's "pinged" attribute
	updateExpression := "SET pinged = list_append(if_not_exists(pinged, :empty_list), :new_ping)"
	expressionAttributeValues := map[string]types.AttributeValue{
		":new_ping":   &types.AttributeValueMemberL{Value: []types.AttributeValue{&types.AttributeValueMemberM{Value: newPing}}},
		":empty_list": &types.AttributeValueMemberL{Value: []types.AttributeValue{}},
	}

	// Define key for the target user's profile
	key := map[string]types.AttributeValue{
		"emailId": &types.AttributeValueMemberS{Value: targetEmailId},
	}

	// Update the target user's profile in DynamoDB
	_, err = as.Dynamo.UpdateItem(ctx, "UserProfiles", updateExpression, key, expressionAttributeValues, nil)
	if err != nil {
		return fmt.Errorf("failed to update target user profile with ping: %w", err)
	}

	return nil
}

// AcceptPing accepts a ping and creates a match
func (as *ActionService) AcceptPing(ctx context.Context, userID string, targetUserID string) error {
	matchID := uuid.NewString()
	currentTime := time.Now().Format(time.RFC3339)

	// Update matches for current user
	currentUserUpdateExpr := "SET matches = list_append(if_not_exists(matches, :empty), :newMatch)"
	_, err := as.Dynamo.UpdateItem(ctx, "UserProfiles", currentUserUpdateExpr, map[string]types.AttributeValue{
		"userId": &types.AttributeValueMemberS{Value: userID},
	}, map[string]types.AttributeValue{
		":empty": &types.AttributeValueMemberL{Value: []types.AttributeValue{}},
		":newMatch": &types.AttributeValueMemberL{Value: []types.AttributeValue{
			&types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
				"matchId": &types.AttributeValueMemberS{Value: matchID},
				"userId":  &types.AttributeValueMemberS{Value: targetUserID},
			}},
		}},
	}, nil)

	if err != nil {
		return fmt.Errorf("failed to update current user matches: %w", err)
	}

	// Update matches for target user
	targetUserUpdateExpr := "SET matches = list_append(if_not_exists(matches, :empty), :newMatch)"
	_, err = as.Dynamo.UpdateItem(ctx, "UserProfiles", targetUserUpdateExpr, map[string]types.AttributeValue{
		"userId": &types.AttributeValueMemberS{Value: targetUserID},
	}, map[string]types.AttributeValue{
		":empty": &types.AttributeValueMemberL{Value: []types.AttributeValue{}},
		":newMatch": &types.AttributeValueMemberL{Value: []types.AttributeValue{
			&types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
				"matchId": &types.AttributeValueMemberS{Value: matchID},
				"userId":  &types.AttributeValueMemberS{Value: userID},
			}},
		}},
	}, nil)

	if err != nil {
		return fmt.Errorf("failed to update target user matches: %w", err)
	}

	// Add a message for the match
	err = as.Dynamo.PutItem(ctx, "Messages", map[string]interface{}{
		"messageId": uuid.NewString(),
		"matchId":   matchID,
		"content":   "It's a match! Start chatting now.",
		"createdAt": currentTime,
	})
	if err != nil {
		return fmt.Errorf("failed to add match message: %w", err)
	}

	return nil
}

// DeclinePing declines a ping
func (as *ActionService) DeclinePing(ctx context.Context, userID string, targetUserID string) error {
	// Add to the notLiked list
	updateExpression := "SET notLiked = list_append(if_not_exists(notLiked, :empty), :targetUserID)"
	_, err := as.Dynamo.UpdateItem(ctx, "UserProfiles", updateExpression, map[string]types.AttributeValue{
		"userId": &types.AttributeValueMemberS{Value: userID},
	}, map[string]types.AttributeValue{
		":empty": &types.AttributeValueMemberL{Value: []types.AttributeValue{}},
		":targetUserID": &types.AttributeValueMemberL{Value: []types.AttributeValue{
			&types.AttributeValueMemberS{Value: targetUserID},
		}},
	}, nil)

	if err != nil {
		return fmt.Errorf("failed to decline ping: %w", err)
	}
	return nil
}

// ProcessAction processes actions like "liked", "notliked", and "pinged"
func (as *ActionService) ProcessAction(ctx context.Context, userID, targetUserID, action, pingNote string) (map[string]string, error) {
	// Fetch the profiles of both users
	currentProfile, err := as.GetUserProfile(ctx, userID)
	if err != nil || currentProfile == nil {
		return nil, errors.New("current user profile not found")
	}

	targetProfile, err := as.GetUserProfile(ctx, targetUserID)
	if err != nil || targetProfile == nil {
		return nil, errors.New("target user profile not found")
	}

	// Handle different actions
	switch action {
	case "liked":
		return as.handleLiked(ctx, userID, targetUserID, targetProfile)
	case "notliked":
		return as.handleNotLiked(ctx, userID, targetUserID)
	case "pinged":
		return as.handlePinged(ctx, userID, targetUserID, pingNote)
	default:
		return nil, errors.New("invalid action")
	}
}

// handleLiked processes the "liked" action
func (as *ActionService) handleLiked(ctx context.Context, userID, targetUserID string, targetProfile map[string]types.AttributeValue) (map[string]string, error) {
	// Check if the action is mutual
	if likedAttr, ok := targetProfile["liked"]; ok {
		likedUsers := likedAttr.(*types.AttributeValueMemberL).Value
		for _, user := range likedUsers {
			if user.(*types.AttributeValueMemberS).Value == userID {
				// Create a match if mutual
				matchID := uuid.NewString()
				if err := as.createMatch(ctx, userID, targetUserID, matchID); err != nil {
					return nil, err
				}
				return map[string]string{"message": "It's a match!", "matchId": matchID}, nil
			}
		}
	}

	// Update the "liked" list
	_, err := as.Dynamo.UpdateItem(ctx, "UserProfiles", "SET liked = list_append(if_not_exists(liked, :empty), :targetUserID)", map[string]types.AttributeValue{
		"userId": &types.AttributeValueMemberS{Value: userID},
	}, map[string]types.AttributeValue{
		":empty":        &types.AttributeValueMemberL{Value: []types.AttributeValue{}},
		":targetUserID": &types.AttributeValueMemberS{Value: targetUserID},
	}, nil)

	if err != nil {
		return nil, fmt.Errorf("failed to update liked list: %w", err)
	}

	return map[string]string{"message": "User liked successfully"}, nil
}

// handleNotLiked processes the "notliked" action
func (as *ActionService) handleNotLiked(ctx context.Context, userID, targetUserID string) (map[string]string, error) {
	// Add to the "notLiked" list
	_, err := as.Dynamo.UpdateItem(ctx, "UserProfiles", "SET notLiked = list_append(if_not_exists(notLiked, :empty), :targetUserID)", map[string]types.AttributeValue{
		"userId": &types.AttributeValueMemberS{Value: userID},
	}, map[string]types.AttributeValue{
		":empty":        &types.AttributeValueMemberL{Value: []types.AttributeValue{}},
		":targetUserID": &types.AttributeValueMemberS{Value: targetUserID},
	}, nil)

	if err != nil {
		return nil, fmt.Errorf("failed to update notLiked list: %w", err)
	}

	return map[string]string{"message": "User added to notLiked list"}, nil
}

// handlePinged processes the "pinged" action
func (as *ActionService) handlePinged(ctx context.Context, userID, targetUserID, pingNote string) (map[string]string, error) {
	if pingNote == "" {
		return nil, errors.New("ping note is required for the 'pinged' action")
	}

	// Update the target user's pings
	_, err := as.Dynamo.UpdateItem(ctx, "UserProfiles", "SET pings = list_append(if_not_exists(pings, :empty), :newPing)", map[string]types.AttributeValue{
		"userId": &types.AttributeValueMemberS{Value: targetUserID},
	}, map[string]types.AttributeValue{
		":empty": &types.AttributeValueMemberL{Value: []types.AttributeValue{}},
		":newPing": &types.AttributeValueMemberL{Value: []types.AttributeValue{
			&types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
				"userId":   &types.AttributeValueMemberS{Value: userID},
				"pingNote": &types.AttributeValueMemberS{Value: pingNote},
			}},
		}},
	}, nil)

	if err != nil {
		return nil, fmt.Errorf("failed to update pings: %w", err)
	}

	return map[string]string{"message": "Ping sent successfully"}, nil
}

// createMatch creates a match between two users
func (as *ActionService) createMatch(ctx context.Context, userID, targetUserID, matchID string) error {
	currentTime := time.Now().Format(time.RFC3339)

	// Update matches for both users
	for _, user := range []string{userID, targetUserID} {
		_, err := as.Dynamo.UpdateItem(ctx, "UserProfiles", "SET matches = list_append(if_not_exists(matches, :empty), :match)", map[string]types.AttributeValue{
			"userId": &types.AttributeValueMemberS{Value: user},
		}, map[string]types.AttributeValue{
			":empty": &types.AttributeValueMemberL{Value: []types.AttributeValue{}},
			":match": &types.AttributeValueMemberL{Value: []types.AttributeValue{
				&types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
					"matchId": &types.AttributeValueMemberS{Value: matchID},
					"userId":  &types.AttributeValueMemberS{Value: targetUserID},
				}},
			}},
		}, nil)
		if err != nil {
			return fmt.Errorf("failed to create match: %w", err)
		}
	}

	// Add a match message
	return as.Dynamo.PutItem(ctx, "Messages", map[string]interface{}{
		"messageId": uuid.NewString(),
		"matchId":   matchID,
		"content":   "It's a match! Start chatting now.",
		"createdAt": currentTime,
	})
}

// extractString safely extracts a string from DynamoDB attribute map
func extractString(profile map[string]types.AttributeValue, field string) string {
	if attr, ok := profile[field]; ok {
		if v, ok := attr.(*types.AttributeValueMemberS); ok {
			return v.Value
		}
	}
	return ""
}

// extractFirstPhoto extracts the first photo URL from the "photos" attribute
func extractFirstPhoto(profile map[string]types.AttributeValue, field string) string {
	if attr, ok := profile[field]; ok {
		if photos, ok := attr.(*types.AttributeValueMemberL); ok && len(photos.Value) > 0 {
			if photo, ok := photos.Value[0].(*types.AttributeValueMemberS); ok {
				return photo.Value
			}
		}
	}
	return ""
}
