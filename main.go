package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

var (
	dynamoClient *dynamodb.Client
)

func init() {
	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(os.Getenv("AWS_REGION")))
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}
	dynamoClient = dynamodb.NewFromConfig(cfg)
}

// Message structure
type Message struct {
	MessageID string    `dynamodbav:"messageId"`
	MatchID   string    `dynamodbav:"matchId"`
	SenderID  *string   `dynamodbav:"senderId"`
	Content   string    `dynamodbav:"content"`
	CreatedAt time.Time `dynamodbav:"createdAt"`
	Liked     bool      `dynamodbav:"liked"`
	Read      bool      `dynamodbav:"read"`
}

// UserProfile structure
type UserProfile struct {
	UserID   string `dynamodbav:"userId"`
	EmailID  string `dynamodbav:"emailId"`
	FullName string `dynamodbav:"fullName"`
}

// AddMessage adds a message to the Messages table
func AddMessage(ctx context.Context, message Message) error {
	if message.MessageID == "" {
		message.MessageID = uuid.NewString()
	}

	item, err := attributevalue.MarshalMap(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	_, err = dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String("Messages"),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(messageId)"),
	})
	if err != nil {
		return fmt.Errorf("failed to add message: %w", err)
	}
	return nil
}

// GetMessages fetches messages by matchId
func GetMessages(ctx context.Context, matchID string, limit int32, lastEvaluatedKey map[string]types.AttributeValue) ([]Message, map[string]types.AttributeValue, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String("Messages"),
		KeyConditionExpression: aws.String("matchId = :matchId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":matchId": &types.AttributeValueMemberS{Value: matchID},
		},
		Limit:             aws.Int32(limit),
		ExclusiveStartKey: lastEvaluatedKey,
		ScanIndexForward:  aws.Bool(false), // Descending order
	}

	output, err := dynamoClient.Query(ctx, input)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query messages: %w", err)
	}

	var messages []Message
	err = attributevalue.UnmarshalListOfMaps(output.Items, &messages)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal messages: %w", err)
	}

	return messages, output.LastEvaluatedKey, nil
}

// AddUserProfile adds a user profile to the UserProfiles table
func AddUserProfile(ctx context.Context, profile UserProfile) error {
	if profile.UserID == "" {
		profile.UserID = uuid.NewString()
	}

	item, err := attributevalue.MarshalMap(profile)
	if err != nil {
		return fmt.Errorf("failed to marshal profile: %w", err)
	}

	_, err = dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String("UserProfiles"),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to add profile: %w", err)
	}
	return nil
}

// GetUserProfile fetches a user profile by userId
func GetUserProfile(ctx context.Context, userID string) (*UserProfile, error) {
	output, err := dynamoClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String("UserProfiles"),
		Key: map[string]types.AttributeValue{
			"userId": &types.AttributeValueMemberS{Value: userID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	if output.Item == nil {
		return nil, errors.New("profile not found")
	}

	var profile UserProfile
	err = attributevalue.UnmarshalMap(output.Item, &profile)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal profile: %w", err)
	}

	return &profile, nil
}

// DeleteUserProfile deletes a user profile by userId
func DeleteUserProfile(ctx context.Context, userID string) error {
	_, err := dynamoClient.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String("UserProfiles"),
		Key: map[string]types.AttributeValue{
			"userId": &types.AttributeValueMemberS{Value: userID},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}
	return nil
}

func main() {
	// Example usage
	ctx := context.TODO()

	// Add a message
	message := Message{
		MatchID:   "123",
		SenderID:  aws.String("456"),
		Content:   "Hello, World!",
		CreatedAt: time.Now(),
		Liked:     false,
		Read:      false,
	}
	err := AddMessage(ctx, message)
	if err != nil {
		log.Fatalf("Failed to add message: %v", err)
	}

	// Fetch messages
	messages, _, err := GetMessages(ctx, "123", 10, nil)
	if err != nil {
		log.Fatalf("Failed to fetch messages: %v", err)
	}
	fmt.Println("Messages:", messages)
}
