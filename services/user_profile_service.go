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

// GetUserProfileByEmail retrieves a user profile by email and calculates distance if needed
func (ups *UserProfileService) GetUserProfileByEmail(ctx context.Context, emailID string, targetEmailID *string) (*models.UserProfile, error) {
	profile, err := ups.GetUserProfileByEmailWithoutDistance(ctx, emailID)
	if err != nil || profile == nil {
		return nil, err
	}

	if targetEmailID == nil || *targetEmailID == "" {
		return profile, nil
	}

	targetProfile, err := ups.GetUserProfileByEmailWithoutDistance(ctx, *targetEmailID)
	if err != nil || targetProfile == nil {
		return profile, nil
	}

	distance := utils.CalculateDistance(profile.Latitude, profile.Longitude, targetProfile.Latitude, targetProfile.Longitude)
	profile.DistanceBetween = math.Round(distance*100) / 100

	return profile, nil
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

// #[TODO] check if below functions are proper
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

// DeleteUserProfile removes a user profile from DynamoDB
func (ups *UserProfileService) DeleteUserProfile(ctx context.Context, userID string) error {
	key := map[string]types.AttributeValue{
		"userId": &types.AttributeValueMemberS{Value: userID},
	}
	return ups.Dynamo.DeleteItem(ctx, models.UserProfilesTable, key)
}

// IsUserHandleAvailable checks if a userhandle is already taken using the GSI
func (ups *UserProfileService) IsUserHandleAvailable(ctx context.Context, userHandle string) (bool, error) {
	log.Printf("🔍 Checking availability of userhandle: %s", userHandle)

	// Query using the Global Secondary Index (GSI) instead of the main table
	keyCondition := "userhandle = :userhandle"
	expressionAttributeValues := map[string]types.AttributeValue{
		":userhandle": &types.AttributeValueMemberS{Value: userHandle},
	}

	// ✅ Specify the GSI name (`userhandle-index`) instead of the main table
	items, err := ups.Dynamo.QueryItemsWithIndex(ctx, models.UserProfilesTable, "userhandle-index", keyCondition, expressionAttributeValues, nil, 1)
	if err != nil {
		log.Printf("❌ Error querying userhandle: %v\n", err)
		return false, fmt.Errorf("failed to check userhandle: %w", err)
	}

	// If userhandle is NOT present in DB, assume it's available
	if len(items) == 0 {
		log.Println("✅ No matching userhandle found, assuming availability.")
		return true, nil
	}

	// Unmarshal the item
	var profile models.UserProfile
	err = attributevalue.UnmarshalMap(items[0], &profile)
	if err != nil {
		log.Printf("❌ Error unmarshalling user profile: %v\n", err)
		return false, fmt.Errorf("failed to unmarshal profile: %w", err)
	}

	// If userhandle exists, return false (taken)
	log.Println("❌ Userhandle is already taken.")
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
