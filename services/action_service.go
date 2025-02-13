package services

import (
	"context"
	"errors"
	"fmt"
	"time"
	"vibin_server/utils"

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
	err = as.removePing(ctx, emailId, targetEmailId)
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
	err = as.removePing(ctx, emailId, targetEmailId)
	if err != nil {
		return fmt.Errorf("failed to remove ping after decline: %w", err)
	}

	return nil
}

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

				// Remove targetEmailId from likedBy[] in emailId's profile
				updateExpressionLikedBy := "REMOVE likedBy[" + as.getListIndex(ctx, emailId, "likedBy", targetEmailId) + "]"
				_, err = as.Dynamo.UpdateItem(ctx, "UserProfiles", updateExpressionLikedBy, map[string]types.AttributeValue{
					"emailId": &types.AttributeValueMemberS{Value: emailId},
				}, nil, nil)
				if err != nil {
					return nil, fmt.Errorf("failed to remove targetEmailId from likedBy[]: %w", err)
				}

				// Remove emailId from liked[] in targetEmailId's profile
				updateExpressionLiked := "REMOVE liked[" + as.getListIndex(ctx, targetEmailId, "liked", emailId) + "]"
				_, err = as.Dynamo.UpdateItem(ctx, "UserProfiles", updateExpressionLiked, map[string]types.AttributeValue{
					"emailId": &types.AttributeValueMemberS{Value: targetEmailId},
				}, nil, nil)
				if err != nil {
					return nil, fmt.Errorf("failed to remove emailId from liked[]: %w", err)
				}

				// Use CreateMessage to insert a welcome message when a match is created
				messageContent := "You have matched with " + utils.ExtractString(targetProfile, "name") + "! Say Hi!"
				err = as.CreateMessage(ctx, matchID, "", messageContent, false, true)
				if err != nil {
					return nil, fmt.Errorf("failed to add match message: %w", err)
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
		":empty": &types.AttributeValueMemberL{Value: []types.AttributeValue{}},
		":targetEmailIdList": &types.AttributeValueMemberL{Value: []types.AttributeValue{
			&types.AttributeValueMemberS{Value: targetEmailId},
		}},
	}

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
		":empty": &types.AttributeValueMemberL{Value: []types.AttributeValue{}},
		":emailIdList": &types.AttributeValueMemberL{Value: []types.AttributeValue{
			&types.AttributeValueMemberS{Value: emailId},
		}},
	}

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

	// Map to store each user’s match data
	matchData := map[string]map[string]types.AttributeValue{
		emailId: {
			"matchId": &types.AttributeValueMemberS{Value: matchID},
			"emailId": &types.AttributeValueMemberS{Value: targetEmailId}, // Store the other user's emailId
		},
		targetEmailId: {
			"matchId": &types.AttributeValueMemberS{Value: matchID},
			"emailId": &types.AttributeValueMemberS{Value: emailId}, // Store the other user's emailId
		},
	}

	// Add match entry for both users
	for user, match := range matchData {
		_, err := as.Dynamo.UpdateItem(ctx, "UserProfiles",
			"SET matches = list_append(if_not_exists(matches, :empty), :newMatch)",
			map[string]types.AttributeValue{
				"emailId": &types.AttributeValueMemberS{Value: user}, // The current user
			},
			map[string]types.AttributeValue{
				":empty": &types.AttributeValueMemberL{Value: []types.AttributeValue{}},
				":newMatch": &types.AttributeValueMemberL{Value: []types.AttributeValue{
					&types.AttributeValueMemberM{Value: match}, // Add the other user's emailId in match
				}},
			}, nil)

		if err != nil {
			return fmt.Errorf("failed to create match for user %s: %w", user, err)
		}
	}

	return nil
}

func (as *ActionService) getListIndex(ctx context.Context, emailId, listName, targetEmailId string) string {
	profile, err := as.GetUserProfile(ctx, emailId)
	if err != nil {
		return ""
	}

	if listAttr, ok := profile[listName]; ok {
		listValues := listAttr.(*types.AttributeValueMemberL).Value
		for i, user := range listValues {
			if user.(*types.AttributeValueMemberS).Value == targetEmailId {
				return fmt.Sprintf("%d", i)
			}
		}
	}
	return ""
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

func (as *ActionService) removePing(ctx context.Context, emailId, senderEmailId string) error {
	// Retrieve the user profile of the current user (who received the ping)
	profile, err := as.GetUserProfile(ctx, emailId)
	if err != nil {
		return fmt.Errorf("failed to fetch user profile: %w", err)
	}

	// Check if pings exist
	if pingsAttr, ok := profile["pings"]; ok {
		pings := pingsAttr.(*types.AttributeValueMemberL).Value
		var updatedPings []types.AttributeValue
		pingRemoved := false

		// Filter out the ping from the sender
		for _, ping := range pings {
			pingMap := ping.(*types.AttributeValueMemberM).Value
			if sender, exists := pingMap["senderEmailId"]; exists && sender.(*types.AttributeValueMemberS).Value == senderEmailId {
				pingRemoved = true
				continue // Skip adding this ping to the new list
			}
			updatedPings = append(updatedPings, ping)
		}

		// If no ping was removed, return early (no need to update)
		if !pingRemoved {
			return nil
		}

		// Construct update expression
		var updateExpression string
		expressionAttributeValues := make(map[string]types.AttributeValue)

		if len(updatedPings) > 0 {
			// If there are remaining pings, update the field
			updateExpression = "SET pings = :updatedPings"
			expressionAttributeValues[":updatedPings"] = &types.AttributeValueMemberL{Value: updatedPings}
		} else {
			// If no pings remain, remove the field
			updateExpression = "REMOVE pings"
		}

		// Update the user profile in DynamoDB (on current user's profile)
		_, err = as.Dynamo.UpdateItem(ctx, "UserProfiles", updateExpression, map[string]types.AttributeValue{
			"emailId": &types.AttributeValueMemberS{Value: emailId},
		}, expressionAttributeValues, nil)

		if err != nil {
			return fmt.Errorf("failed to update pings list: %w", err)
		}
	}

	return nil
}
