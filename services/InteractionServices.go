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
	existingInteraction, err := s.GetInteraction(ctx, sender, receiver, interactionType)
	if err != nil {
		log.Printf("⚠️ Error fetching interaction: %v", err)
		return err
	}

	// Determine status updates based on action
	var newStatus string
	var matchID *string

	switch action {
	case models.InteractionTypeLike:
		if existingInteraction == nil {
			newStatus = models.StatusPending
		} else if existingInteraction.Status == models.StatusPending {
			newStatus = models.StatusMatch
			generatedMatchID := s.GenerateMatchID()
			matchID = &generatedMatchID
		}
	case models.InteractionTypeDislike:
		newStatus = models.StatusDeclined
	case models.InteractionTypePing:
		newStatus = models.StatusPending
	case models.StatusApproved:
		newStatus = models.StatusApproved
		generatedMatchID := s.GenerateMatchID()
		matchID = &generatedMatchID
	case models.StatusRejected:
		newStatus = models.StatusRejected
	default:
		return fmt.Errorf("❌ Unsupported interaction type: %s", interactionType)
	}

	// If it's a new interaction, insert it
	if existingInteraction == nil {
		return s.CreateInteraction(ctx, sender, receiver, interactionType, newStatus, matchID, message)
	}

	// Otherwise, update existing interaction
	return s.UpdateInteractionStatus(ctx, existingInteraction.InteractionID, newStatus, matchID, message)
}

// CreateInteraction inserts a new interaction into DynamoDB
func (s *InteractionService) CreateInteraction(ctx context.Context, sender, receiver, interactionType, status string, matchID *string, message *string) error {
	interactionID := uuid.New().String()
	now := time.Now().Format(time.RFC3339)
	interaction := models.Interaction{
		InteractionID:   interactionID,
		Users:           []string{sender, receiver},
		UserLookup:      sender, // Used for querying
		SenderHandle:    sender,
		InteractionType: interactionType,
		ChatType:        "private",
		Status:          status,
		MatchID:         matchID,
		Message:         message, // ✅ Store message if provided
		CreatedAt:       now,
		LastUpdated:     now,
	}

	log.Printf("📥 Saving new interaction: %+v", interaction)
	return s.Dynamo.PutItem(ctx, models.InteractionsTable, interaction)
}

// UpdateInteractionStatus updates the status of an existing interaction
func (s *InteractionService) UpdateInteractionStatus(ctx context.Context, interactionID, newStatus string, matchID *string, message *string) error {
	log.Printf("🔄 Updating interaction %s to status: %s", interactionID, newStatus)

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

	// Add message if provided (only for pings)
	if message != nil {
		updateExpression += ", #message = :message"
		expressionValues[":message"] = &types.AttributeValueMemberS{Value: *message}
		expressionNames["#message"] = "message"
	}

	// Define key for update
	key := map[string]types.AttributeValue{
		"interactionId": &types.AttributeValueMemberS{Value: interactionID},
	}

	_, err := s.Dynamo.UpdateItem(ctx, models.InteractionsTable, updateExpression, key, expressionValues, expressionNames)
	return err
}

// GetInteraction fetches an interaction between two users
func (s *InteractionService) GetInteraction(ctx context.Context, sender, receiver, interactionType string) (*models.Interaction, error) {
	log.Printf("🔍 Checking if interaction exists: %s -> %s (%s)", sender, receiver, interactionType)

	keyCondition := "userLookup = :user"
	filterExpression := "interactionType = :interactionType"
	expressionValues := map[string]types.AttributeValue{
		":user":            &types.AttributeValueMemberS{Value: sender},
		":interactionType": &types.AttributeValueMemberS{Value: interactionType},
	}

	items, err := s.Dynamo.QueryItemsWithIndex(ctx, models.InteractionsTable, models.UserLookupIndex, keyCondition, expressionValues, map[string]string{"FilterExpression": filterExpression}, 1)
	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		return nil, nil // No interaction found
	}

	var interaction models.Interaction
	err = attributevalue.UnmarshalMap(items[0], &interaction)
	if err != nil {
		return nil, err
	}

	return &interaction, nil
}

// GenerateMatchID generates a new UUID for matches
func (s *InteractionService) GenerateMatchID() string {
	return uuid.New().String()
}

// GetUserInteractions fetches all interactions involving a specific user
func (s *InteractionService) GetUserInteractions(ctx context.Context, userHandle string) ([]models.Interaction, error) {
	log.Printf("🔍 Fetching interactions for user: %s", userHandle)

	keyCondition := "userLookup = :user"
	expressionValues := map[string]types.AttributeValue{
		":user": &types.AttributeValueMemberS{Value: userHandle},
	}

	items, err := s.Dynamo.QueryItemsWithIndex(ctx, models.InteractionsTable, models.UserLookupIndex, keyCondition, expressionValues, nil, 100)
	if err != nil {
		return nil, err
	}

	var interactions []models.Interaction
	err = attributevalue.UnmarshalListOfMaps(items, &interactions)
	if err != nil {
		return nil, err
	}

	return interactions, nil
}

// ✅ GetInteractedUsers retrieves users who have interacted (liked/disliked)
func (s *InteractionService) GetInteractedUsers(ctx context.Context, userHandle string, interactionTypes []string) (map[string]bool, error) {
	log.Printf("🔍 Fetching interactions of types %v for user: %s", interactionTypes, userHandle)

	// 🔹 Step 1: Use `userLookup` for querying instead of `users`
	keyCondition := "userLookup = :user"
	expressionAttributeValues := map[string]types.AttributeValue{
		":user": &types.AttributeValueMemberS{Value: userHandle},
	}

	// 🔹 Step 2: Query the `users-index` (which now uses `userLookup` as PK)
	items, err := s.Dynamo.QueryItemsWithIndex(ctx, models.InteractionsTable, models.UserLookupIndex, keyCondition, expressionAttributeValues, nil, 100)
	if err != nil {
		log.Printf("❌ Error querying interactions: %v", err)
		return nil, fmt.Errorf("failed to fetch interactions: %w", err)
	}

	// 🔹 Step 3: Filter interactions based on interactionTypes
	interactedUsers := make(map[string]bool)
	for _, item := range items {
		var interaction models.Interaction
		err := attributevalue.UnmarshalMap(item, &interaction)
		if err != nil {
			continue
		}

		// 🔹 Only include interactions matching the specified types
		if contains(interactionTypes, interaction.InteractionType) {
			for _, user := range interaction.Users {
				if user != userHandle { // ✅ Store only the other user
					interactedUsers[user] = true
				}
			}
		}
	}

	log.Printf("✅ Found %d interacted users for %s", len(interactedUsers), userHandle)
	return interactedUsers, nil
}

// ✅ contains checks if a slice contains a specific value
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}
