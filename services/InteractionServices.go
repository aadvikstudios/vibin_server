package services

import (
	"context"
	"fmt"
	"log"
	"strings"
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

// GetInteraction retrieves an interaction between two users
func (s *InteractionService) GetInteraction(ctx context.Context, sender, receiver string) (*models.Interaction, error) {
	log.Printf("🔍 Checking if interaction exists: %s -> %s", sender, receiver)

	key := map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: "USER#" + sender},
		"SK": &types.AttributeValueMemberS{Value: "INTERACTION#" + receiver},
	}

	item, err := s.Dynamo.GetItem(ctx, models.InteractionsTable, key)
	if err != nil {
		log.Printf("❌ DynamoDB error while fetching interaction: %v", err)
		return nil, err
	}
	if item == nil {
		// Do not log this as an error, since it's expected for new users
		log.Printf("ℹ️ No previous interaction found for %s -> %s. Proceeding to create a new one.", sender, receiver)
		return nil, nil
	}

	var interaction models.Interaction
	err = attributevalue.UnmarshalMap(item, &interaction)
	if err != nil {
		log.Printf("❌ Error unmarshalling interaction: %v", err)
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

	items, err := s.Dynamo.QueryItems(ctx, models.InteractionsTable, keyCondition, expressionValues, nil, 100)
	if err != nil {
		log.Printf("❌ Error querying interactions: %v", err)
		return nil, fmt.Errorf("failed to fetch interactions: %w", err)
	}

	var interactions []models.Interaction
	err = attributevalue.UnmarshalListOfMaps(items, &interactions)
	if err != nil {
		log.Printf("❌ Error unmarshaling interactions: %v", err)
		return nil, fmt.Errorf("failed to process data: %w", err)
	}

	log.Printf("✅ Found %d interactions for %s", len(interactions), userHandle)
	return interactions, nil
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

	// 🔥 If the interaction does not exist, create it
	if existingInteraction == nil {
		log.Printf("🆕 Interaction does not exist. Creating new interaction...")
		return s.CreateInteraction(ctx, sender, receiver, interactionType, newStatus, matchID, message)
	}

	// 🔥 Otherwise, update existing interaction
	return s.UpdateInteractionStatus(ctx, sender, receiver, newStatus, matchID, message)
}

// CreateInteraction inserts a new interaction into DynamoDB
func (s *InteractionService) CreateInteraction(ctx context.Context, sender, receiver, interactionType, status string, matchID *string, message *string) error {
	log.Printf("🆕 Creating a new interaction for %s -> %s", sender, receiver)

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
	err := s.Dynamo.PutItem(ctx, models.InteractionsTable, interaction)
	if err != nil {
		log.Printf("❌ Error inserting interaction: %v", err)
		return fmt.Errorf("failed to create interaction: %w", err)
	}
	log.Println("✅ Interaction successfully created.")
	return nil
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

// GetMutualMatches fetches all mutual matches for a user
func (s *InteractionService) GetMutualMatches(ctx context.Context, userHandle string) ([]string, error) {
	log.Printf("🔍 Fetching mutual matches for user: %s", userHandle)

	keyCondition := "PK = :user"
	expressionValues := map[string]types.AttributeValue{
		":user": &types.AttributeValueMemberS{Value: "USER#" + userHandle},
	}

	filterExpression := "#status = :match"
	expressionValues[":match"] = &types.AttributeValueMemberS{Value: "match"}
	expressionNames := map[string]string{"#status": "status"}

	items, err := s.Dynamo.QueryItemsWithFilters(ctx, models.InteractionsTable, keyCondition, expressionValues, expressionNames, filterExpression)
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

// GetInteractedUsers retrieves users who have interacted (liked, pinged, matched) with a specific user
func (s *InteractionService) GetInteractedUsers(ctx context.Context, userHandle string, interactionTypes []string) ([]string, error) {
	log.Printf("🔍 Fetching interacted users for: %s with types: %v", userHandle, interactionTypes)

	keyCondition := "PK = :user"
	expressionValues := map[string]types.AttributeValue{
		":user": &types.AttributeValueMemberS{Value: "USER#" + userHandle},
	}

	var filterExpressions []string
	expressionNames := map[string]string{"#interactionType": "interactionType"}

	for i, interactionType := range interactionTypes {
		paramName := fmt.Sprintf(":interactionType%d", i)
		expressionValues[paramName] = &types.AttributeValueMemberS{Value: interactionType}
		filterExpressions = append(filterExpressions, fmt.Sprintf("#interactionType = %s", paramName))
	}

	filterExpression := strings.Join(filterExpressions, " OR ")

	items, err := s.Dynamo.QueryItemsWithFilters(ctx, models.InteractionsTable, keyCondition, expressionValues, expressionNames, filterExpression)
	if err != nil {
		log.Printf("❌ Error querying interactions: %v", err)
		return nil, fmt.Errorf("failed to fetch interacted users: %w", err)
	}

	var users []string
	for _, item := range items {
		var interaction models.Interaction
		err := attributevalue.UnmarshalMap(item, &interaction)
		if err == nil {
			users = append(users, interaction.ReceiverHandle)
		}
	}

	log.Printf("✅ Found %d interacted users for %s", len(users), userHandle)
	return users, nil
}
