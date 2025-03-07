package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"vibin_server/models"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type UserProfileService struct {
	Dynamo *DynamoService
}

// AddUserProfile adds a new user profile to DynamoDB
func (ups *UserProfileService) AddUserProfile(ctx context.Context, profile models.UserProfile) (*models.UserProfile, error) {
	err := ups.Dynamo.PutItem(ctx, models.UserProfilesTable, profile)
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

// GetUserProfile retrieves a user profile by ID
func (ups *UserProfileService) GetUserProfile(ctx context.Context, emailID string) (*models.UserProfile, error) {
	key := map[string]types.AttributeValue{
		"emailId": &types.AttributeValueMemberS{Value: emailID},
	}

	item, err := ups.Dynamo.GetItem(ctx, models.UserProfilesTable, key)
	if err != nil {
		return nil, err
	}

	if item == nil {
		return nil, errors.New("profile not found")
	}

	var profile models.UserProfile
	err = attributevalue.UnmarshalMap(item, &profile)
	if err != nil {
		return nil, err
	}

	return &profile, nil
}

// GetUserProfileByEmail fetches a user profile based on the email GSI (`emailId-index`)
func (ups *UserProfileService) GetUserProfileByEmail(ctx context.Context, emailID string) (*models.UserProfile, error) {
	log.Printf("🔍 Fetching user profile for email: %s", emailID)

	// Define query parameters for the GSI (emailId-index)
	keyCondition := "emailId = :emailId"
	expressionAttributeValues := map[string]types.AttributeValue{
		":emailId": &types.AttributeValueMemberS{Value: emailID},
	}

	// Query the GSI (emailId-index)
	items, err := ups.Dynamo.QueryItemsWithIndex(ctx, models.UserProfilesTable, "emailId-index", keyCondition, expressionAttributeValues, nil, 1)
	if err != nil {
		log.Printf("❌ Error querying email index: %v", err)
		return nil, fmt.Errorf("failed to fetch profile by email: %w", err)
	}

	// If no profile is found, return nil
	if len(items) == 0 {
		log.Printf("❌ No profile found for email: %s", emailID)
		return nil, nil
	}

	// Unmarshal the first result into a UserProfile struct
	var profile models.UserProfile
	err = attributevalue.UnmarshalMap(items[0], &profile)
	if err != nil {
		log.Printf("❌ Error unmarshalling user profile: %v", err)
		return nil, fmt.Errorf("failed to unmarshal profile: %w", err)
	}

	log.Printf("✅ Successfully fetched user profile: %+v", profile)
	return &profile, nil
}

// UpdateUserProfile updates an existing user profile
func (ups *UserProfileService) UpdateUserProfile(ctx context.Context, emailID string, updates map[string]interface{}) (*models.UserProfile, error) {
	key := map[string]types.AttributeValue{
		"emailId": &types.AttributeValueMemberS{Value: emailID},
	}

	// Construct UpdateExpression, ExpressionAttributeValues, and ExpressionAttributeNames
	updateExpression := "SET"
	expressionAttributeValues := make(map[string]types.AttributeValue)
	expressionAttributeNames := make(map[string]string)

	for field, value := range updates {
		placeholder := ":" + field
		attributeName := "#" + field
		updateExpression += " " + attributeName + " = " + placeholder + ","

		// Convert value dynamically
		switch v := value.(type) {
		case string:
			expressionAttributeValues[placeholder] = &types.AttributeValueMemberS{Value: v}
		case bool:
			expressionAttributeValues[placeholder] = &types.AttributeValueMemberBOOL{Value: v}
		case int:
			expressionAttributeValues[placeholder] = &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", v)}
		case float64:
			expressionAttributeValues[placeholder] = &types.AttributeValueMemberN{Value: fmt.Sprintf("%f", v)}
		case []string:
			stringSlice, _ := attributevalue.MarshalList(v)
			expressionAttributeValues[placeholder] = &types.AttributeValueMemberL{Value: stringSlice}
		default:
			return nil, fmt.Errorf("unsupported update type for field %s", field)
		}

		expressionAttributeNames[attributeName] = field
	}

	// Remove trailing comma
	updateExpression = updateExpression[:len(updateExpression)-1]

	// Call UpdateItem with correctly formatted parameters
	updatedItem, err := ups.Dynamo.UpdateItem(ctx, models.UserProfilesTable, updateExpression, key, expressionAttributeValues, expressionAttributeNames)
	if err != nil {
		return nil, err
	}

	// Unmarshal response
	var updatedProfile models.UserProfile
	err = attributevalue.UnmarshalMap(updatedItem, &updatedProfile)
	if err != nil {
		return nil, err
	}

	return &updatedProfile, nil
}

// DeleteUserProfile removes a user profile from DynamoDB
func (ups *UserProfileService) DeleteUserProfile(ctx context.Context, userID string) error {
	key := map[string]types.AttributeValue{
		"userId": &types.AttributeValueMemberS{Value: userID},
	}
	return ups.Dynamo.DeleteItem(ctx, models.UserProfilesTable, key)
}

func (ups *UserProfileService) IsUserHandleAvailable(ctx context.Context, userHandle string) (bool, error) {
	log.Printf("🔍 Checking availability of userhandle: %s", userHandle)

	// Define the partition key for lookup
	key := map[string]types.AttributeValue{
		"userhandle": &types.AttributeValueMemberS{Value: userHandle},
	}

	// Fetch item using GetItem
	item, err := ups.Dynamo.GetItem(ctx, models.UserProfilesTable, key)
	if err != nil {
		// ✅ Check if error contains "item not found"
		if strings.Contains(err.Error(), "item not found") {
			log.Printf("✅ Userhandle '%s' is available (not found in DynamoDB).", userHandle)
			return true, nil
		}

		// ❌ Unexpected errors should still be logged and returned
		log.Printf("❌ Unexpected error retrieving userhandle '%s' from DynamoDB: %v", userHandle, err)
		return false, fmt.Errorf("failed to check userhandle: %w", err)
	}

	// If no item is returned, the userhandle is available
	if item == nil || len(item) == 0 {
		log.Printf("✅ Userhandle '%s' is available.", userHandle)
		return true, nil
	}

	// ❌ Userhandle exists, return false
	log.Printf("❌ Userhandle '%s' is already taken.", userHandle)
	return false, nil
}

// CheckEmailExists checks if an email ID exists in the database
func (ups *UserProfileService) CheckEmailExists(ctx context.Context, emailID string) (bool, error) {
	log.Printf("🔍 Checking if email exists: %s", emailID)

	// Define query parameters
	keyCondition := "emailId = :emailId"
	expressionAttributeValues := map[string]types.AttributeValue{
		":emailId": &types.AttributeValueMemberS{Value: emailID},
	}

	// Query GSI (emailId-index)
	items, err := ups.Dynamo.QueryItemsWithIndex(ctx, models.UserProfilesTable, "emailId-index", keyCondition, expressionAttributeValues, nil, 1)
	if err != nil {
		log.Printf("❌ Error querying email index: %v", err)
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}

	// If items found, email exists
	exists := len(items) > 0
	log.Printf("✅ Email found: %t", exists)
	return exists, nil
}

// GetUserHandleByEmail retrieves a userhandle based on an email lookup
func (ups *UserProfileService) GetUserHandleByEmail(ctx context.Context, emailID string) (string, error) {
	log.Printf("🔍 Fetching userhandle for email: %s", emailID)

	// Define query parameters
	keyCondition := "emailId = :emailId"
	expressionAttributeValues := map[string]types.AttributeValue{
		":emailId": &types.AttributeValueMemberS{Value: emailID},
	}

	// Query GSI (emailId-index)
	items, err := ups.Dynamo.QueryItemsWithIndex(ctx, models.UserProfilesTable, "emailId-index", keyCondition, expressionAttributeValues, nil, 1)
	if err != nil {
		log.Printf("❌ Error querying email index: %v", err)
		return "", fmt.Errorf("failed to fetch userhandle: %w", err)
	}

	// If no item found, return 404
	if len(items) == 0 {
		log.Printf("❌ Email not found: %s", emailID)
		return "", nil
	}

	// Unmarshal and extract userhandle
	var profile models.UserProfile
	err = attributevalue.UnmarshalMap(items[0], &profile)
	if err != nil {
		log.Printf("❌ Error unmarshalling user profile: %v", err)
		return "", fmt.Errorf("failed to unmarshal user profile: %w", err)
	}

	log.Printf("✅ Found userhandle: %s for email: %s", profile.UserHandle, emailID)
	return profile.UserHandle, nil
}

// ✅ Haversine formula to calculate distance (in km) between two lat/lng points
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Earth's radius in km
	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLon := (lon2 - lon1) * (math.Pi / 180.0)

	lat1Rad := lat1 * (math.Pi / 180.0)
	lat2Rad := lat2 * (math.Pi / 180.0)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1Rad)*math.Cos(lat2Rad)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

// GetUserSuggestions retrieves a list of users based on gender & interaction history
func (ups *UserProfileService) GetUserSuggestions(ctx context.Context, userHandle, gender string) ([]models.UserProfile, error) {
	log.Printf("🔍 Fetching user suggestions for gender: %s, excluding interactions from: %s", gender, userHandle)

	// Step 1: Fetch the requester's latitude & longitude
	requesterProfile, err := ups.GetUserProfileByHandle(ctx, userHandle)
	if err != nil {
		log.Printf("❌ Error fetching requester profile: %v", err)
		return nil, fmt.Errorf("failed to fetch requester profile: %w", err)
	}

	if requesterProfile.Latitude == 0 || requesterProfile.Longitude == 0 {
		log.Println("⚠️ Requester profile does not have valid latitude/longitude")
		return nil, fmt.Errorf("requester location missing")
	}

	// Step 2: Fetch interaction history (liked/disliked profiles)
	interactionService := InteractionService{Dynamo: ups.Dynamo} // Use InteractionService
	interactedUsersList, err := interactionService.GetInteractedUsers(ctx, userHandle, []string{models.InteractionTypeLike, models.InteractionTypeDislike})
	if err != nil {
		log.Printf("❌ Error fetching interaction history: %v", err)
		return nil, fmt.Errorf("failed to fetch interactions: %w", err)
	}

	// Convert interactedUsersList (slice) to a map for quick lookups
	interactedUsers := make(map[string]bool)
	for _, user := range interactedUsersList {
		interactedUsers[user] = true
	}

	// Step 3: Query the `gender-index` GSI to get potential matches
	keyCondition := "gender = :gender"
	expressionAttributeValues := map[string]types.AttributeValue{
		":gender": &types.AttributeValueMemberS{Value: gender},
	}

	items, err := ups.Dynamo.QueryItemsWithIndex(ctx, models.UserProfilesTable, "gender-index", keyCondition, expressionAttributeValues, nil, 50)
	if err != nil {
		log.Printf("❌ Error querying gender index: %v", err)
		return nil, fmt.Errorf("failed to fetch user suggestions: %w", err)
	}

	if len(items) == 0 {
		log.Println("⚠️ No profiles found matching the criteria.")
		return []models.UserProfile{}, nil
	}

	// Step 4: Unmarshal result into a list of UserProfile structs
	var profiles []models.UserProfile
	err = attributevalue.UnmarshalListOfMaps(items, &profiles)
	if err != nil {
		log.Printf("❌ Error unmarshalling user profiles: %v", err)
		return nil, fmt.Errorf("failed to unmarshal user profiles: %w", err)
	}

	// Step 5: Filter out users who are already liked/disliked & calculate distance
	filteredProfiles := make([]models.UserProfile, 0)
	for _, profile := range profiles {
		// Exclude self & users without valid location
		if profile.UserHandle != userHandle && profile.Latitude != 0 && profile.Longitude != 0 {
			if _, exists := interactedUsers[profile.UserHandle]; !exists { // ✅ Skip already interacted users
				profile.DistanceBetween = haversine(requesterProfile.Latitude, requesterProfile.Longitude, profile.Latitude, profile.Longitude)
				filteredProfiles = append(filteredProfiles, profile)
			}
		}
	}

	// Step 6: Sort by distance (nearest first)
	sort.Slice(filteredProfiles, func(i, j int) bool {
		return filteredProfiles[i].DistanceBetween < filteredProfiles[j].DistanceBetween
	})

	log.Printf("✅ Successfully fetched %d user suggestions.", len(filteredProfiles))
	return filteredProfiles, nil
}

// ✅ Fetch a user profile by userHandle
func (ups *UserProfileService) GetUserProfileByHandle(ctx context.Context, userHandle string) (*models.UserProfile, error) {
	key := map[string]types.AttributeValue{
		"userhandle": &types.AttributeValueMemberS{Value: userHandle},
	}

	item, err := ups.Dynamo.GetItem(ctx, models.UserProfilesTable, key)
	if err != nil {
		return nil, err
	}

	if item == nil {
		return nil, fmt.Errorf("profile not found")
	}

	var profile models.UserProfile
	err = attributevalue.UnmarshalMap(item, &profile)
	if err != nil {
		return nil, err
	}

	return &profile, nil
}
