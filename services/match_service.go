package services

import (
	"context"
	"fmt"
	"vibin_server/models"
	"vibin_server/utils"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type MatchService struct {
	Dynamo *DynamoService
}

// GetUserProfile retrieves a user profile by ID
func (as *MatchService) GetUserProfile(ctx context.Context, emailId string) (map[string]types.AttributeValue, error) {
	key := map[string]types.AttributeValue{
		"emailId": &types.AttributeValueMemberS{Value: emailId},
	}
	return as.Dynamo.GetItem(ctx, "UserProfiles", key)
}

func (as *MatchService) GetPings(ctx context.Context, emailId string) ([]map[string]interface{}, error) {
	profile, err := as.GetUserProfile(ctx, emailId)
	if err != nil || profile == nil {
		return nil, fmt.Errorf("user profile not found for emailId: %s", emailId)
	}

	pingsAttr, ok := profile["pings"]
	if !ok {
		return []map[string]interface{}{}, nil
	}

	pings := pingsAttr.(*types.AttributeValueMemberL).Value
	var enrichedPings []map[string]interface{}

	for _, ping := range pings {
		pingData, ok := ping.(*types.AttributeValueMemberM)
		if !ok {
			continue
		}

		senderEmailId := utils.ExtractString(pingData.Value, "senderEmailId")
		pingNote := utils.ExtractString(pingData.Value, "pingNote")

		senderProfile, err := as.GetUserProfile(ctx, senderEmailId)
		if err != nil {
			continue
		}

		enrichedPings = append(enrichedPings, map[string]interface{}{
			"senderEmailId":     senderEmailId,
			"senderName":        utils.ExtractString(senderProfile, "name"),
			"senderGender":      utils.ExtractString(senderProfile, "gender"),
			"senderPhoto":       utils.ExtractPhotoURLs(senderProfile), // Use the new function
			"senderOrientation": utils.ExtractString(senderProfile, "orientation"),
			"senderAge":         utils.ExtractInt(senderProfile, "age"),
			"pingNote":          pingNote,
		})
	}

	return enrichedPings, nil
}

// GetCurrentMatches retrieves the matches for a user and enriches them with messages data.
func (as *MatchService) GetCurrentMatches(ctx context.Context, emailId string) ([]map[string]interface{}, error) {
	profile, err := as.GetUserProfile(ctx, emailId)
	if err != nil || profile == nil {
		return nil, fmt.Errorf("user profile not found for userId: %s", emailId)
	}

	matchesAttr, ok := profile["matches"]
	if !ok {
		return []map[string]interface{}{}, nil
	}

	matches := matchesAttr.(*types.AttributeValueMemberL).Value
	var matchedProfiles []map[string]interface{}

	for _, match := range matches {
		matchData := match.(*types.AttributeValueMemberM).Value
		matchUserID := matchData["emailId"].(*types.AttributeValueMemberS).Value

		// Fetch target profile details
		targetProfile, err := as.GetUserProfile(ctx, matchUserID)
		if err != nil {
			continue
		}

		// Extract necessary fields
		name := utils.ExtractString(targetProfile, "name")
		photo := utils.ExtractFirstPhoto(targetProfile, "photos") // First photo URL
		matchID := utils.ExtractString(matchData, "matchId")

		// Fetch latest message details
		lastMessage, isUnread, senderId := as.GetLastMessage(ctx, matchID, emailId)

		// Append to matchedProfiles
		matchedProfiles = append(matchedProfiles, map[string]interface{}{
			"matchId":     matchID,
			"emailId":     matchUserID,
			"name":        name,
			"photo":       photo,
			"lastMessage": lastMessage,
			"isUnread":    isUnread,
			"senderId":    senderId,
		})
	}

	return matchedProfiles, nil
}

func (as *MatchService) GetNewLikes(ctx context.Context, userID string) ([]map[string]interface{}, error) {
	profile, err := as.GetUserProfile(ctx, userID)
	if err != nil || profile == nil {
		return nil, fmt.Errorf("user profile not found for userId: %s", userID)
	}

	likedByAttr, ok := profile["likedBy"]
	if !ok {
		return []map[string]interface{}{}, nil
	}

	likedBy := likedByAttr.(*types.AttributeValueMemberL).Value
	var likedProfiles []map[string]interface{}

	for _, liked := range likedBy {
		likedEmailId := liked.(*types.AttributeValueMemberS).Value

		likedProfile, err := as.GetUserProfile(ctx, likedEmailId)
		if err != nil {
			continue
		}

		likedProfiles = append(likedProfiles, map[string]interface{}{
			"emailId": likedEmailId,
			"name":    utils.ExtractString(likedProfile, "name"),
			"photos":  utils.ExtractPhotoURLs(likedProfile), // Use the new function
		})
	}

	return likedProfiles, nil
}

func (as *MatchService) GetFilteredProfiles(
	ctx context.Context,
	emailId, gender string,
	additionalFilters map[string]string,
) ([]models.UserProfile, error) {
	// Fetch the user's profile to get liked, notLiked, and matches
	userProfile, err := as.GetUserProfile(ctx, emailId)
	if err != nil || userProfile == nil {
		return nil, fmt.Errorf("failed to fetch user profile for emailId: %s", emailId)
	}

	// Prepare exclusion filters
	excludeEmails := map[string]struct{}{}
	excludeEmails[emailId] = struct{}{} // Exclude the user themselves

	// Add liked[] to exclusion
	if likedAttr, ok := userProfile["liked"]; ok {
		for _, liked := range likedAttr.(*types.AttributeValueMemberL).Value {
			excludeEmails[liked.(*types.AttributeValueMemberS).Value] = struct{}{}
		}
	}

	// Add notLiked[] to exclusion
	if notLikedAttr, ok := userProfile["notLiked"]; ok {
		for _, notLiked := range notLikedAttr.(*types.AttributeValueMemberL).Value {
			excludeEmails[notLiked.(*types.AttributeValueMemberS).Value] = struct{}{}
		}
	}

	// Add matches[] to exclusion
	if matchesAttr, ok := userProfile["matches"]; ok {
		for _, match := range matchesAttr.(*types.AttributeValueMemberL).Value {
			matchData := match.(*types.AttributeValueMemberM).Value
			matchEmailId := matchData["emailId"].(*types.AttributeValueMemberS).Value
			excludeEmails[matchEmailId] = struct{}{}
		}
	}

	// Build filters for DynamoDB scan
	excludeFields := map[string]string{
		"gender": gender, // Exclude same gender
	}

	// Merge additional filters
	for key, value := range additionalFilters {
		excludeFields[key] = value
	}

	// Use DynamoService to scan profiles with filters
	var profiles []models.UserProfile
	err = as.Dynamo.ScanWithFilter(ctx, models.UserProfilesTable, func(profile map[string]types.AttributeValue) bool {
		// Exclude profiles based on emailId
		emailId := profile["emailId"].(*types.AttributeValueMemberS).Value
		if _, excluded := excludeEmails[emailId]; excluded {
			return false
		}
		return true
	}, excludeFields, &profiles)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch filtered profiles: %w", err)
	}

	return profiles, nil
}

// GetLastMessage fetches the latest message for a match
func (as *MatchService) GetLastMessage(ctx context.Context, matchID, emailId string) (string, bool, string) {
	// Query DynamoDB to get the latest message for this match
	filterExpression := "matchId = :matchId"
	expressionAttributeValues := map[string]types.AttributeValue{
		":matchId": &types.AttributeValueMemberS{Value: matchID},
	}

	// Query the Messages table sorted by timestamp
	messages, err := as.Dynamo.QueryItems(ctx, "Messages", filterExpression, expressionAttributeValues, nil, 1)
	if err != nil || len(messages) == 0 {
		return "", false, "" // No messages found
	}

	// Extract last message details
	messageItem := messages[0]
	lastMessage := utils.ExtractString(messageItem, "content")
	senderId := utils.ExtractString(messageItem, "senderId")
	isUnread := utils.ExtractBool(messageItem, "isUnread") && senderId != emailId // Only unread if sender is not the user

	return lastMessage, isUnread, senderId
}
