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

	newPing := &types.AttributeValueMemberM{
		Value: map[string]types.AttributeValue{
			"senderEmailId": &types.AttributeValueMemberS{Value: emailId},
			"pingNote":      &types.AttributeValueMemberS{Value: pingNote},
			"createdAt":     &types.AttributeValueMemberS{Value: createdAt},
		},
	}

	// Add the ping to the target user's "pings" list
	if err := as.AddToList(ctx, targetEmailId, "pings", newPing); err != nil {
		return fmt.Errorf("failed to send ping: %w", err)
	}

	return nil
}

// ProcessPingAction processes "accept" or "decline" ping actions
func (as *ActionService) ProcessPingAction(ctx context.Context, emailId, targetEmailId, action, pingNote string) (map[string]string, error) {
	switch action {
	case "accept":
		return as.AcceptPing(ctx, emailId, targetEmailId, pingNote)
	case "decline":
		if err := as.DeclinePing(ctx, emailId, targetEmailId); err != nil {
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
	if err := as.createMatch(ctx, emailId, targetEmailId, matchID); err != nil {
		return nil, fmt.Errorf("failed to create match: %w", err)
	}

	// Add match ID to both users' "matches" lists
	if err := as.AddToList(ctx, emailId, "matches", &types.AttributeValueMemberS{Value: matchID}); err != nil {
		return nil, fmt.Errorf("failed to update matches list for %s: %w", emailId, err)
	}
	if err := as.AddToList(ctx, targetEmailId, "matches", &types.AttributeValueMemberS{Value: matchID}); err != nil {
		return nil, fmt.Errorf("failed to update matches list for %s: %w", targetEmailId, err)
	}

	// Send a message for the match
	if err := as.CreateMessage(ctx, matchID, targetEmailId, pingNote, false, true); err != nil {
		return nil, fmt.Errorf("failed to add match message: %w", err)
	}

	// Remove the ping after acceptance
	if err := as.RemoveFromList(ctx, emailId, "pings", targetEmailId); err != nil {
		return nil, fmt.Errorf("failed to remove ping after acceptance: %w", err)
	}

	return map[string]string{"message": "It's a match!", "matchId": matchID}, nil
}

// DeclinePing declines a ping request
func (as *ActionService) DeclinePing(ctx context.Context, emailId, targetEmailId string) error {
	// Add targetEmailId to the "notLiked" list
	if err := as.AddToList(ctx, emailId, "notLiked", &types.AttributeValueMemberS{Value: targetEmailId}); err != nil {
		return fmt.Errorf("failed to add to notLiked list: %w", err)
	}

	// Remove targetEmailId from the "pings" list
	if err := as.RemoveFromList(ctx, emailId, "pings", targetEmailId); err != nil {
		return fmt.Errorf("failed to remove from pings list: %w", err)
	}

	return nil
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

func (as *ActionService) handleLiked(ctx context.Context, emailId, targetEmailId string) (map[string]string, error) {
	// Fetch the target user's profile
	targetProfile, err := as.GetUserProfile(ctx, targetEmailId)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch target user profile: %w", err)
	}

	// Check if the target user has already liked this user
	if as.IsMutualLike(targetProfile, emailId) {
		// Create a match if mutual like exists
		matchID := uuid.NewString()
		if err := as.createMatch(ctx, emailId, targetEmailId, matchID); err != nil {
			return nil, err
		}

		// Remove mutual likes from both users
		if err := as.RemoveMutualLikes(ctx, emailId, targetEmailId); err != nil {
			return nil, err
		}

		// Send a match message
		messageContent := fmt.Sprintf("You have matched with %s! Say Hi!", as.ExtractName(targetProfile))
		if err := as.CreateMessage(ctx, matchID, "", messageContent, false, true); err != nil {
			return nil, fmt.Errorf("failed to add match message: %w", err)
		}

		return map[string]string{"message": "It's a match!", "matchId": matchID}, nil
	}

	// Add targetEmailId to the "liked" list of the emailId user
	if err := as.AddToList(ctx, emailId, "liked", &types.AttributeValueMemberS{Value: targetEmailId}); err != nil {
		return nil, fmt.Errorf("failed to update liked list for emailId: %w", err)
	}

	// Add emailId to the "likedBy" list of the targetEmailId user
	if err := as.AddToList(ctx, targetEmailId, "likedBy", &types.AttributeValueMemberS{Value: emailId}); err != nil {
		return nil, fmt.Errorf("failed to update likedBy list for targetEmailId: %w", err)
	}

	return map[string]string{"message": "User liked successfully"}, nil
}
func (as *ActionService) handleNotLiked(ctx context.Context, emailId, targetEmailId string) (map[string]string, error) {
	// Add targetEmailId to the "notLiked" list
	if err := as.AddToList(ctx, emailId, "notLiked", &types.AttributeValueMemberS{Value: targetEmailId}); err != nil {
		return nil, fmt.Errorf("failed to update notLiked list: %w", err)
	}

	return map[string]string{"message": "User added to notLiked list"}, nil
}

func (as *ActionService) IsMutualLike(targetProfile map[string]types.AttributeValue, emailId string) bool {
	if likedAttr, ok := targetProfile["liked"]; ok {
		likedUsers := likedAttr.(*types.AttributeValueMemberL).Value
		for _, user := range likedUsers {
			if user.(*types.AttributeValueMemberS).Value == emailId {
				return true
			}
		}
	}
	return false
}
func (as *ActionService) RemoveMutualLikes(ctx context.Context, emailId, targetEmailId string) error {
	if err := as.RemoveFromList(ctx, emailId, "likedBy", targetEmailId); err != nil {
		return fmt.Errorf("failed to remove targetEmailId from likedBy list: %w", err)
	}

	if err := as.RemoveFromList(ctx, targetEmailId, "liked", emailId); err != nil {
		return fmt.Errorf("failed to remove emailId from liked list: %w", err)
	}

	return nil
}

func (as *ActionService) ExtractName(profile map[string]types.AttributeValue) string {
	if nameAttr, ok := profile["name"]; ok {
		if name, ok := nameAttr.(*types.AttributeValueMemberS); ok {
			return name.Value
		}
	}
	return "Unknown"
}

// CreateMatch creates a match entry for two users
func (as *ActionService) createMatch(ctx context.Context, emailId, targetEmailId, matchID string) error {
	// Match entry for `emailId` (stores `targetEmailId`)
	matchEntryA := map[string]types.AttributeValue{
		"matchId": &types.AttributeValueMemberS{Value: matchID},
		"emailId": &types.AttributeValueMemberS{Value: targetEmailId},
	}

	// Match entry for `targetEmailId` (stores `emailId`)
	matchEntryB := map[string]types.AttributeValue{
		"matchId": &types.AttributeValueMemberS{Value: matchID},
		"emailId": &types.AttributeValueMemberS{Value: emailId},
	}

	// Add match entry for both users
	if err := as.AddToList(ctx, emailId, "matches", &types.AttributeValueMemberM{Value: matchEntryA}); err != nil {
		return fmt.Errorf("failed to add match for %s: %w", emailId, err)
	}
	if err := as.AddToList(ctx, targetEmailId, "matches", &types.AttributeValueMemberM{Value: matchEntryB}); err != nil {
		return fmt.Errorf("failed to add match for %s: %w", targetEmailId, err)
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

	if err := as.Dynamo.PutItem(ctx, "Messages", message); err != nil {
		return fmt.Errorf("failed to add message: %w", err)
	}
	return nil
}

// AddToList updates a user's list attribute (e.g., "pings", "matches", "notLiked") by appending a new value.
func (as *ActionService) AddToList(ctx context.Context, userProfileEmail, attribute string, value types.AttributeValue) error {
	updateExpression := fmt.Sprintf("SET %s = list_append(if_not_exists(%s, :empty), :newItem)", attribute, attribute)

	_, err := as.Dynamo.UpdateItem(ctx, "UserProfiles", updateExpression,
		map[string]types.AttributeValue{"emailId": &types.AttributeValueMemberS{Value: userProfileEmail}},
		map[string]types.AttributeValue{
			":empty":   &types.AttributeValueMemberL{Value: []types.AttributeValue{}},
			":newItem": &types.AttributeValueMemberL{Value: []types.AttributeValue{value}},
		}, nil,
	)

	if err != nil {
		return fmt.Errorf("failed to add item to %s list: %w", attribute, err)
	}

	return nil
}

func (as *ActionService) RemoveFromList(ctx context.Context, userProfileEmail, attribute, emailIdToRemove string) error {
	profile, err := as.GetUserProfile(ctx, userProfileEmail)
	if err != nil {
		return fmt.Errorf("failed to fetch user profile: %w", err)
	}

	// Check if the list attribute exists
	listAttr, exists := profile[attribute]
	if !exists {
		return fmt.Errorf("list '%s' not found", attribute)
	}

	listValues, ok := listAttr.(*types.AttributeValueMemberL)
	if !ok || len(listValues.Value) == 0 {
		return fmt.Errorf("list '%s' is empty", attribute)
	}

	// Find the index of the item to remove
	var itemIndex int = -1
	for i, item := range listValues.Value {
		if email, ok := item.(*types.AttributeValueMemberS); ok && email.Value == emailIdToRemove {
			itemIndex = i
			break
		}
	}

	// If item is not found, return without making an unnecessary update
	if itemIndex == -1 {
		return fmt.Errorf("email '%s' not found in list '%s'", emailIdToRemove, attribute)
	}

	// Construct REMOVE expression
	updateExpression := fmt.Sprintf("REMOVE %s[%d]", attribute, itemIndex)

	_, err = as.Dynamo.UpdateItem(ctx, "UserProfiles", updateExpression,
		map[string]types.AttributeValue{"emailId": &types.AttributeValueMemberS{Value: userProfileEmail}}, nil, nil,
	)

	if err != nil {
		return fmt.Errorf("failed to remove email from %s list: %w", attribute, err)
	}

	return nil
}
