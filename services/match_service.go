package services

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"vibin_server/models"
	"vibin_server/utils"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/aws"
)

type MatchService struct {
	Dynamo *DynamoService
}

// Haversine formula to calculate distance between two coordinates in km
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Earth's radius in km
	lat1Rad := lat1 * (math.Pi / 180)
	lon1Rad := lon1 * (math.Pi / 180)
	lat2Rad := lat2 * (math.Pi / 180)
	lon2Rad := lon2 * (math.Pi / 180)

	deltaLat := lat2Rad - lat1Rad
	deltaLon := lon2Rad - lon1Rad

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
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
func (as *MatchService) GetConnections(ctx context.Context, emailId string) ([]map[string]interface{}, error) {
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
	// Fetch the user's profile to get latitude and longitude
	userProfile, err := as.GetUserProfile(ctx, emailId)
	if err != nil || userProfile == nil {
		return nil, fmt.Errorf("failed to fetch user profile for emailId: %s", emailId)
	}

	// Extract user location
	userLat := userProfile["latitude"].(*types.AttributeValueMemberN).Value
	userLon := userProfile["longitude"].(*types.AttributeValueMemberN).Value
	currentLat := parseFloat(userLat)
	currentLon := parseFloat(userLon)

	// Prepare exclusion filters
	excludeEmails := map[string]struct{}{emailId: {}}

	// Add liked, notLiked, and matches[] to exclusion
	if likedAttr, ok := userProfile["liked"]; ok {
		for _, liked := range likedAttr.(*types.AttributeValueMemberL).Value {
			excludeEmails[liked.(*types.AttributeValueMemberS).Value] = struct{}{}
		}
	}
	if notLikedAttr, ok := userProfile["notLiked"]; ok {
		for _, notLiked := range notLikedAttr.(*types.AttributeValueMemberL).Value {
			excludeEmails[notLiked.(*types.AttributeValueMemberS).Value] = struct{}{}
		}
	}
	if matchesAttr, ok := userProfile["matches"]; ok {
		for _, match := range matchesAttr.(*types.AttributeValueMemberL).Value {
			matchData := match.(*types.AttributeValueMemberM).Value
			matchEmailId := matchData["emailId"].(*types.AttributeValueMemberS).Value
			excludeEmails[matchEmailId] = struct{}{}
		}
	}

	// Build filters for DynamoDB scan
	excludeFields := map[string]string{"gender": gender}
	for key, value := range additionalFilters {
		excludeFields[key] = value
	}

	// Use DynamoDB scan with filters
	var profiles []models.UserProfile
	err = as.Dynamo.ScanWithFilter(ctx, models.UserProfilesTable, func(profile map[string]types.AttributeValue) bool {
		email := profile["emailId"].(*types.AttributeValueMemberS).Value
		if _, excluded := excludeEmails[email]; excluded {
			return false
		}
		return true
	}, excludeFields, &profiles)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch filtered profiles: %w", err)
	}
	fmt.Printf("✅ Fetched profiles length before iterating %d", len(profiles))
	// Compute distances and attach to each profile
	for i := range profiles {
		profile := &profiles[i]

		// Extract lat/lon from profile
		profileLat := profile.Latitude
		profileLon := profile.Longitude

		// Ensure profile has valid lat/lon before calculating distance
		if profileLat == 0 || profileLon == 0 {
			fmt.Printf("⚠️ Skipping distance calculation for profile %d: Missing lat/lon\n", i)
			continue
		}

		// Calculate distance
		distance := calculateDistance(currentLat, currentLon, profileLat, profileLon)
		profile.DistanceBetween = math.Round(distance*100) / 100

		// Ensure zero distances are not omitted
		if profile.DistanceBetween == 0 {
			profile.DistanceBetween = 0.00
		}

		// Debug log for distance
		fmt.Printf("✅ Distance calculated for profile %d (%s): %f km\n", i, profile.EmailID, profile.DistanceBetween)
	}
	// Return filtered and sorted profiles
	return profiles, nil
}

// Helper function to parse float values safely
func parseFloat(value string) float64 {
	val, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0.0
	}
	return val
}

// GetLastMessage fetches the latest message for a match
func (as *MatchService) GetLastMessage(ctx context.Context, matchID, emailId string) (string, bool, string) {
	// Query DynamoDB to get the latest message for this match
	filterExpression := "matchId = :matchId"
	expressionAttributeValues := map[string]types.AttributeValue{
		":matchId": &types.AttributeValueMemberS{Value: matchID},
	}

	// Add ScanIndexForward: false to ensure the query is sorted by timestamp descending
	queryInput := &dynamodb.QueryInput{
		TableName:                 aws.String("Messages"),
		KeyConditionExpression:    aws.String(filterExpression),
		ExpressionAttributeValues: expressionAttributeValues,
		ScanIndexForward:          aws.Bool(false), // Descending order
		Limit:                     aws.Int32(1),    // Only get the latest message
	}

	// Execute the query
	queryOutput, err := as.Dynamo.QueryItemsWithQueryInput(ctx, queryInput)
	if err != nil || len(queryOutput) == 0 {
		return "", false, "" // No messages found
	}

	// Extract the latest message details
	messageItem := queryOutput[0]
	lastMessage := utils.ExtractString(messageItem, "content")
	senderId := utils.ExtractString(messageItem, "senderId")
	isUnread := utils.ExtractBool(messageItem, "isUnread") && senderId != emailId // Only unread if sender is not the user

	return lastMessage, isUnread, senderId
}
