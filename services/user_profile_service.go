package services

import (
	"context"
	"errors"
	"fmt"
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

// GetUserProfileByEmail retrieves a user profile by email using a GSI
func (ups *UserProfileService) GetUserProfileByEmail(ctx context.Context, emailID string) (*models.UserProfile, error) {
	keyCondition := "emailId = :emailId"
	expressionAttributeValues := map[string]types.AttributeValue{
		":emailId": &types.AttributeValueMemberS{Value: emailID},
	}

	items, err := ups.Dynamo.QueryItems(ctx, models.UserProfilesTable, keyCondition, expressionAttributeValues, nil, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch profile by email: %w", err)
	}

	if len(items) == 0 {
		return nil, errors.New("profile not found")
	}

	var profile models.UserProfile
	err = attributevalue.UnmarshalMap(items[0], &profile)
	if err != nil {
		return nil, err
	}

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
