package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type UserProfile struct {
	UserID              string   `dynamodbav:"userId"`
	FullName            string   `dynamodbav:"fullName"`
	EmailID             string   `dynamodbav:"emailId"`
	Bio                 string   `dynamodbav:"bio"`
	Desires             []string `dynamodbav:"desires"`
	DOB                 string   `dynamodbav:"dob"`
	Gender              string   `dynamodbav:"gender"`
	Interests           []string `dynamodbav:"interests"`
	Latitude            float64  `dynamodbav:"latitude"`
	Longitude           float64  `dynamodbav:"longitude"`
	LookingFor          string   `dynamodbav:"lookingFor"`
	Orientation         string   `dynamodbav:"orientation"`
	ShowGenderOnProfile bool     `dynamodbav:"showGenderOnProfile"`
}

const UserProfilesTable = "UserProfiles"

type UserProfileService struct {
	Dynamo *DynamoService
}

// AddUserProfile adds a new user profile to DynamoDB
func (ups *UserProfileService) AddUserProfile(ctx context.Context, profile UserProfile) (*UserProfile, error) {
	err := ups.Dynamo.PutItem(ctx, UserProfilesTable, profile)
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

// GetUserProfile retrieves a user profile by ID
func (ups *UserProfileService) GetUserProfile(ctx context.Context, userID string) (*UserProfile, error) {
	key := map[string]types.AttributeValue{
		"userId": &types.AttributeValueMemberS{Value: userID},
	}

	item, err := ups.Dynamo.GetItem(ctx, UserProfilesTable, key)
	if err != nil {
		return nil, err
	}

	if item == nil {
		return nil, errors.New("profile not found")
	}

	var profile UserProfile
	err = attributevalue.UnmarshalMap(item, &profile)
	if err != nil {
		return nil, err
	}

	return &profile, nil
}

// GetUserProfileByEmail retrieves a user profile by email using a GSI (Global Secondary Index)
func (ups *UserProfileService) GetUserProfileByEmail(ctx context.Context, emailID string) (*UserProfile, error) {
	keyCondition := "emailId = :emailId"
	expressionAttributeValues := map[string]types.AttributeValue{
		":emailId": &types.AttributeValueMemberS{Value: emailID},
	}

	items, err := ups.Dynamo.QueryItems(ctx, UserProfilesTable, keyCondition, expressionAttributeValues, nil, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch profile by email: %w", err)
	}

	if len(items) == 0 {
		return nil, errors.New("profile not found")
	}

	var profile UserProfile
	err = attributevalue.UnmarshalMap(items[0], &profile)
	if err != nil {
		return nil, err
	}

	return &profile, nil
}

// UpdateUserProfile updates an existing user profile
func (ups *UserProfileService) UpdateUserProfile(ctx context.Context, userID string, updates map[string]interface{}) (*UserProfile, error) {
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

	updatedItem, err := ups.Dynamo.UpdateItem(ctx, UserProfilesTable, updateExpression, key, expressionAttributeValues, expressionAttributeNames)
	if err != nil {
		return nil, err
	}

	var updatedProfile UserProfile
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
	return ups.Dynamo.DeleteItem(ctx, UserProfilesTable, key)
}
