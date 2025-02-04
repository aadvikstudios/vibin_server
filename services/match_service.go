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

// GetPings retrieves the pings for a user and enriches them with sender details
func (as *MatchService) GetPings(ctx context.Context, emailId string) ([]map[string]interface{}, error) {
	// Fetch the user's profile
	profile, err := as.GetUserProfile(ctx, emailId)
	if err != nil || profile == nil {
		return nil, fmt.Errorf("user profile not found for emailId: %s", emailId)
	}

	// Retrieve "pings" attribute
	pingsAttr, ok := profile["pings"]
	if !ok {
		return []map[string]interface{}{}, nil // No pings, return an empty array
	}

	pings := pingsAttr.(*types.AttributeValueMemberL).Value
	var enrichedPings []map[string]interface{}

	// Iterate through each ping, fetch sender profile, and enrich data
	for _, ping := range pings {
		pingData, ok := ping.(*types.AttributeValueMemberM)
		if !ok {
			continue
		}

		// Extract sender email ID from the ping data
		senderEmailId := utils.ExtractString(pingData.Value, "senderEmailId")
		pingNote := utils.ExtractString(pingData.Value, "pingNote")

		// Fetch sender's profile
		senderProfile, err := as.GetUserProfile(ctx, senderEmailId)
		if err != nil {
			continue // Skip if sender profile is not found
		}

		// Extract sender details from the sender's profile
		senderName := utils.ExtractString(senderProfile, "name")
		senderGender := utils.ExtractString(senderProfile, "gender")
		senderOrientation := utils.ExtractString(senderProfile, "orientation")
		senderAge := utils.ExtractInt(senderProfile, "age")
		senderPhoto := utils.ExtractFirstPhoto(senderProfile, "photos")

		// Append enriched ping data
		enrichedPings = append(enrichedPings, map[string]interface{}{
			"senderEmailId":     senderEmailId,
			"senderName":        senderName,
			"senderGender":      senderGender,
			"senderPhoto":       senderPhoto,
			"senderOrientation": senderOrientation,
			"senderAge":         senderAge,
			"pingNote":          pingNote,
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
		likedEmailId := liked.(*types.AttributeValueMemberS).Value

		likedProfile, err := as.GetUserProfile(ctx, likedEmailId)
		if err != nil {
			continue
		}

		// Extract photos as a slice of strings
		photoURLs := []string{}
		if photosAttr, ok := likedProfile["photos"]; ok {
			if photos, ok := photosAttr.(*types.AttributeValueMemberL); ok {
				for _, photo := range photos.Value {
					if photoURL, ok := photo.(*types.AttributeValueMemberS); ok {
						photoURLs = append(photoURLs, photoURL.Value)
					}
				}
			}
		}

		// Append enriched profile data
		likedProfiles = append(likedProfiles, map[string]interface{}{
			"emailId": likedEmailId,
			"name":    likedProfile["name"].(*types.AttributeValueMemberS).Value,
			"photos":  photoURLs, // Photos as a slice of strings
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
