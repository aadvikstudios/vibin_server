package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"vibin_server/models"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

// InteractionService handles interactions (like, ping, and matches)
type InteractionService struct {
	Dynamo *DynamoService
}

// CreateOrUpdateInteraction handles likes, dislikes, pings, and approvals
func (s *InteractionService) CreateOrUpdateInteraction(ctx context.Context, sender, receiver, interactionType, action string, message *string) error {
	log.Printf("🔄 Processing %s from %s -> %s", interactionType, sender, receiver)

	// Check if an existing interaction exists
	existingInteraction, err := s.GetInteraction(ctx, sender, receiver)
	if err != nil {
		log.Printf("⚠️ Error fetching interaction: %v", err)
		return err
	}

	// Determine status updates based on action
	var newStatus string
	var matchID *string

	switch action {
	case "like":
		newStatus = "pending"
		// Check if User B also liked User A (Mutual Match)
		mutualLike, _ := s.GetInteraction(ctx, receiver, sender)
		if mutualLike != nil && mutualLike.Status == "pending" {
			newStatus = "match"
			generatedMatchID := uuid.New().String()
			matchID = &generatedMatchID
		}
	case "dislike":
		newStatus = "declined"
	case "ping":
		newStatus = "pending"
	case "approve":
		newStatus = "match"
		generatedMatchID := uuid.New().String()
		matchID = &generatedMatchID
	case "reject":
		newStatus = "rejected"
	default:
		return fmt.Errorf("❌ Unsupported interaction type: %s", interactionType)
	}

	// If it's a new interaction, insert it
	if existingInteraction == nil {
		return s.CreateInteraction(ctx, sender, receiver, interactionType, newStatus, matchID, message)
	}

	// Otherwise, update existing interaction
	return s.UpdateInteractionStatus(ctx, sender, receiver, newStatus, matchID, message)
}

// CreateInteraction inserts a new interaction into DynamoDB
func (s *InteractionService) CreateInteraction(ctx context.Context, sender, receiver, interactionType, status string, matchID *string, message *string) error {
	now := time.Now().Format(time.RFC3339)
	interaction := models.Interaction{
		PK:              "USER#" + sender,
		SK:              "INTERACTION#" + receiver,
		SenderHandle:    sender,
		ReceiverHandle:  receiver,
		InteractionType: interactionType,
		Status:          status,
		MatchID:         matchID,
		Message:         message,
		CreatedAt:       now,
		LastUpdated:     now,
	}

	log.Printf("📥 Saving new interaction: %+v", interaction)
	return s.Dynamo.PutItem(ctx, models.InteractionsTable, interaction)
}

// UpdateInteractionStatus updates the status of an existing interaction
func (s *InteractionService) UpdateInteractionStatus(ctx context.Context, sender, receiver, newStatus string, matchID *string, message *string) error {
	log.Printf("🔄 Updating interaction %s -> %s to status: %s", sender, receiver, newStatus)

	updateExpression := "SET #status = :status, #lastUpdated = :lastUpdated"
	expressionValues := map[string]types.AttributeValue{
		":status":      &types.AttributeValueMemberS{Value: newStatus},
		":lastUpdated": &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
	}
	expressionNames := map[string]string{
		"#status":      "status",
		"#lastUpdated": "lastUpdated",
	}

	// Add MatchID if provided
	if matchID != nil {
		updateExpression += ", #matchId = :matchId"
		expressionValues[":matchId"] = &types.AttributeValueMemberS{Value: *matchID}
		expressionNames["#matchId"] = "matchId"
	}

	// Add message if provided
	if message != nil {
		updateExpression += ", #message = :message"
		expressionValues[":message"] = &types.AttributeValueMemberS{Value: *message}
		expressionNames["#message"] = "message"
	}

	// Define key for update
	key := map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: "USER#" + sender},
		"SK": &types.AttributeValueMemberS{Value: "INTERACTION#" + receiver},
	}

	_, err := s.Dynamo.UpdateItem(ctx, models.InteractionsTable, updateExpression, key, expressionValues, expressionNames)
	return err
}

// GetInteraction retrieves an interaction between two users
func (s *InteractionService) GetInteraction(ctx context.Context, sender, receiver string) (*models.Interaction, error) {
	log.Printf("🔍 Checking if interaction exists: %s -> %s", sender, receiver)

	key := map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: "USER#" + sender},
		"SK": &types.AttributeValueMemberS{Value: "INTERACTION#" + receiver},
	}

	item, err := s.Dynamo.GetItem(ctx, models.InteractionsTable, key)
	if err != nil || item == nil {
		return nil, nil
	}

	var interaction models.Interaction
	err = attributevalue.UnmarshalMap(item, &interaction)
	if err != nil {
		return nil, err
	}

	return &interaction, nil
}

// GetUserInteractions fetches all interactions involving a specific user
func (s *InteractionService) GetUserInteractions(ctx context.Context, userHandle string) ([]models.Interaction, error) {
	log.Printf("🔍 Fetching interactions for user: %s", userHandle)

	keyCondition := "PK = :user"
	expressionValues := map[string]types.AttributeValue{
		":user": &types.AttributeValueMemberS{Value: "USER#" + userHandle},
	}

	// Pass additional parameters: empty map for attribute names and limit (e.g., 100 items)
	items, err := s.Dynamo.QueryItems(ctx, models.InteractionsTable, keyCondition, expressionValues, nil, 100)
	if err != nil {
		log.Printf("❌ Error querying interactions: %v", err)
		return nil, fmt.Errorf("failed to fetch interactions: %w", err)
	}

	// Unmarshal items into Go struct
	var interactions []models.Interaction
	err = attributevalue.UnmarshalListOfMaps(items, &interactions)
	if err != nil {
		log.Printf("❌ Error unmarshaling interactions: %v", err)
		return nil, fmt.Errorf("failed to process data: %w", err)
	}

	log.Printf("✅ Found %d interactions for %s", len(interactions), userHandle)
	return interactions, nil
}

// GetMutualMatches fetches all mutual matches for a user
func (s *InteractionService) GetMutualMatches(ctx context.Context, userHandle string) ([]string, error) {
	log.Printf("🔍 Fetching mutual matches for user: %s", userHandle)

	keyCondition := "PK = :user AND #status = :match"
	expressionValues := map[string]types.AttributeValue{
		":user":  &types.AttributeValueMemberS{Value: "USER#" + userHandle},
		":match": &types.AttributeValueMemberS{Value: "match"},
	}
	expressionNames := map[string]string{
		"#status": "status",
	}

	// Query for matches where user is the sender
	items, err := s.Dynamo.QueryItemsWithFilters(ctx, models.InteractionsTable, keyCondition, expressionValues, expressionNames)
	if err != nil {
		log.Printf("❌ Error fetching mutual matches: %v", err)
		return nil, fmt.Errorf("failed to fetch matches: %w", err)
	}

	// Extract matched user handles
	matches := []string{}
	for _, item := range items {
		var interaction models.Interaction
		err := attributevalue.UnmarshalMap(item, &interaction)
		if err != nil {
			continue
		}
		matches = append(matches, interaction.ReceiverHandle)
	}

	log.Printf("✅ Found %d matches for %s", len(matches), userHandle)
	return matches, nil
}
