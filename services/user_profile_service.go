package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"vibin_server/models"
	"vibin_server/utils"

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
func (ups *UserProfileService) GetUserProfile(ctx context.Context, userID string) (*models.UserProfile, error) {
	key := map[string]types.AttributeValue{
		"userId": &types.AttributeValueMemberS{Value: userID},
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
func (ups *UserProfileService) GetUserProfileByEmail(ctx context.Context, emailID string, targetEmailID *string) (*models.UserProfile, error) {
	log.Printf("Fetching profile for email: %s\n", emailID)

	// Fetch the main user profile
	profile, err := ups.GetUserProfileByEmailWithoutDistance(ctx, emailID)

	// Handle case where profile is not found
	if err != nil {
		log.Printf("Error fetching profile: %v\n", err)
		return nil, fmt.Errorf("failed to fetch profile: %w", err)
	}
	if profile == nil {
		log.Printf("❌ Profile not found for email: %s", emailID)
		return nil, nil // 🚀 Return nil so the controller can handle the 404 response
	}

	// If no targetEmailID is provided, return only the profile
	if targetEmailID == nil || *targetEmailID == "" {
		log.Printf("Returning profile without distance calculation (no target email provided).")
		return profile, nil
	}

	// Fetch the target profile for distance calculation
	targetProfile, err := ups.GetUserProfileByEmailWithoutDistance(ctx, *targetEmailID)
	if err != nil || targetProfile == nil {
		log.Printf("Error fetching target profile: %v\n", err)
		return nil, fmt.Errorf("failed to fetch target profile: %w", err)
	}

	// Ensure both profiles have valid latitude and longitude
	if profile.Latitude == 0 || profile.Longitude == 0 || targetProfile.Latitude == 0 || targetProfile.Longitude == 0 {
		log.Printf("⚠️ One or both profiles missing latitude/longitude, skipping distance calculation")
		return profile, nil
	}

	// Calculate distance between the two users
	distance := utils.CalculateDistance(profile.Latitude, profile.Longitude, targetProfile.Latitude, targetProfile.Longitude)

	// Attach the distance to the profile
	profile.DistanceBetween = math.Round(distance*100) / 100 // Round to 2 decimal places

	log.Printf("✅ Distance calculated between %s and %s: %.2f km\n", emailID, *targetEmailID, profile.DistanceBetween)
	return profile, nil
}

// Helper function to fetch a profile by email WITHOUT distance calculation
func (ups *UserProfileService) GetUserProfileByEmailWithoutDistance(ctx context.Context, emailID string) (*models.UserProfile, error) {
	log.Printf("Fetching profile by email: %s\n", emailID)

	keyCondition := "emailId = :emailId"
	expressionAttributeValues := map[string]types.AttributeValue{
		":emailId": &types.AttributeValueMemberS{Value: emailID},
	}

	items, err := ups.Dynamo.QueryItems(ctx, models.UserProfilesTable, keyCondition, expressionAttributeValues, nil, 1)
	if err != nil {
		log.Printf("Error querying DynamoDB: %v\n", err)
		return nil, fmt.Errorf("failed to fetch profile by email: %w", err)
	}

	if len(items) == 0 {
		log.Printf("No profile found for email: %s\n", emailID)
		return nil, nil // No profile found
	}

	var profile models.UserProfile
	err = attributevalue.UnmarshalMap(items[0], &profile)
	if err != nil {
		log.Printf("Error unmarshalling DynamoDB item: %v\n", err)
		return nil, fmt.Errorf("failed to unmarshal profile: %w", err)
	}

	log.Printf("Profile fetched successfully: %+v\n", profile)
	return &profile, nil
}

// UpdateUserProfile updates an existing user profile
func (ups *UserProfileService) UpdateUserProfile(ctx context.Context, userID string, updates map[string]interface{}) (*models.UserProfile, error) {
	key := map[string]types.AttributeValue{
		"userId": &types.AttributeValueMemberS{Value: userID},
	}

	updateExpression := "SET"
	expressionAttributeValues := make(map[string]types.AttributeValue)
	expressionAttributeNames := make(map[string]string)

	for k, v := range updates {
		placeholder := ":" + k
		attributeName := "#" + k
		updateExpression += " " + attributeName + " = " + placeholder + ","

		expressionAttributeValues[placeholder] = &types.AttributeValueMemberS{Value: v.(string)}
		expressionAttributeNames[attributeName] = k
	}

	updateExpression = updateExpression[:len(updateExpression)-1]

	updatedItem, err := ups.Dynamo.UpdateItem(ctx, models.UserProfilesTable, updateExpression, key, expressionAttributeValues, expressionAttributeNames)
	if err != nil {
		return nil, err
	}

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

// ClearUserInteractions removes `liked[]`, `notLiked[]`, `pings[]`, and `matches[]` from a user profile
func (ups *UserProfileService) ClearUserInteractions(emailId string) error {
	log.Printf("🔄 Clearing interactions for user: %s", emailId)

	// Define the REMOVE update expression
	updateExpression := "REMOVE liked, notLiked, pings, matches"

	// Prepare key for the update operation
	key := map[string]types.AttributeValue{
		"emailId": &types.AttributeValueMemberS{Value: emailId},
	}

	// Call the DynamoDB service method
	_, err := ups.Dynamo.UpdateItem(context.TODO(), "UserProfiles", updateExpression, key, nil, nil)
	if err != nil {
		log.Printf("❌ Error clearing interactions for %s: %v", emailId, err)
		return fmt.Errorf("failed to clear user interactions: %w", err)
	}

	log.Printf("✅ Successfully cleared interactions for user: %s", emailId)
	return nil
}
