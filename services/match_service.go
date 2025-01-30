package services

import (
	"context"
	"fmt"
	"vibin_server/models"

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

// GetPings retrieves the pings for a user
func (as *MatchService) GetPings(ctx context.Context, emailId string) ([]map[string]interface{}, error) {
	profile, err := as.GetUserProfile(ctx, emailId)
	if err != nil || profile == nil {
		return nil, fmt.Errorf("user profile not found for userId: %s", emailId)
	}

	pingsAttr, ok := profile["pings"]
	if !ok {
		return []map[string]interface{}{}, nil // No pings, return an empty array
	}

	pings := pingsAttr.(*types.AttributeValueMemberL).Value
	var enrichedPings []map[string]interface{}

	// Enrich each ping with user data
	for _, ping := range pings {
		pingData := ping.(*types.AttributeValueMemberM).Value
		enrichedPings = append(enrichedPings, map[string]interface{}{
			"senderEmailId": pingData["senderEmailId"].(*types.AttributeValueMemberS).Value,
			"pingNote":      pingData["pingNote"].(*types.AttributeValueMemberS).Value,
			"senderName":    pingData["name"].(*types.AttributeValueMemberS).Value,
			"senderPhoto":   pingData["photo"].(*types.AttributeValueMemberS).Value,
			"gender":        pingData["gender"].(*types.AttributeValueMemberS).Value,
		})
	}

	return enrichedPings, nil
}

// GetCurrentMatches retrieves the matches for a user
func (as *MatchService) GetCurrentMatches(ctx context.Context, emailId string) ([]map[string]interface{}, error) {
	profile, err := as.GetUserProfile(ctx, emailId)
	if err != nil || profile == nil {
		return nil, fmt.Errorf("user profile not found for userId: %s", emailId)
	}

	matchesAttr, ok := profile["matches"]
	if !ok {
		return []map[string]interface{}{}, nil // No matches, return an empty array
	}

	matches := matchesAttr.(*types.AttributeValueMemberL).Value
	var matchedProfiles []map[string]interface{}

	// Fetch and enrich each match profile
	for _, match := range matches {
		matchData := match.(*types.AttributeValueMemberM).Value
		matchUserID := matchData["emailId"].(*types.AttributeValueMemberS).Value

		targetProfile, err := as.GetUserProfile(ctx, matchUserID)
		if err != nil {
			continue
		}

		matchedProfiles = append(matchedProfiles, map[string]interface{}{
			"emailId": matchUserID,
			"name":    targetProfile["name"].(*types.AttributeValueMemberS).Value,
			"photos":  targetProfile["photos"].(*types.AttributeValueMemberL).Value,
		})
	}

	return matchedProfiles, nil
}

// GetNewLikes retrieves new likes for a user
func (as *MatchService) GetNewLikes(ctx context.Context, userID string) ([]map[string]interface{}, error) {
	profile, err := as.GetUserProfile(ctx, userID)
	if err != nil || profile == nil {
		return nil, fmt.Errorf("user profile not found for userId: %s", userID)
	}

	likedByAttr, ok := profile["likedBy"]
	if !ok {
		return []map[string]interface{}{}, nil // No likes, return an empty array
	}

	likedBy := likedByAttr.(*types.AttributeValueMemberL).Value
	var likedProfiles []map[string]interface{}

	// Fetch and enrich profiles for each "likedBy" user
	for _, liked := range likedBy {
		likedUserID := liked.(*types.AttributeValueMemberS).Value

		likedProfile, err := as.GetUserProfile(ctx, likedUserID)
		if err != nil {
			continue
		}

		likedProfiles = append(likedProfiles, map[string]interface{}{
			"userId": likedUserID,
			"name":   likedProfile["name"].(*types.AttributeValueMemberS).Value,
			"photos": likedProfile["photos"].(*types.AttributeValueMemberL).Value,
		})
	}

	return likedProfiles, nil
}

func (as *MatchService) GetFilteredProfiles(
	ctx context.Context,
	emailId, gender string,
	additionalFilters map[string]string,
) ([]models.UserProfile, error) {
	// Prepare filters and exclusions
	excludeFields := map[string]string{
		"emailId": emailId,
		"gender":  gender,
	}

	// Merge additional filters
	for key, value := range additionalFilters {
		excludeFields[key] = value
	}

	// Prepare result slice
	var profiles []models.UserProfile

	// Use DynamoService to scan with filters
	if err := as.Dynamo.ScanWithFilter(ctx, models.UserProfilesTable, nil, excludeFields, &profiles); err != nil {
		return nil, fmt.Errorf("failed to fetch filtered profiles: %w", err)
	}

	return profiles, nil
}
