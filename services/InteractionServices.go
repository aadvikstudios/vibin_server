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
		if strings.Contains(err.Error(), "item not found") {
			log.Printf("ℹ️ No previous interaction found for %s -> %s. Proceeding to create a new one.", sender, receiver)
			return nil, nil // ✅ This is expected; allow creation of a new interaction
		}
		log.Printf("❌ Unexpected DynamoDB error while fetching interaction: %v", err)
		return nil, err
	}

	if item == nil {
		log.Printf("ℹ️ No interaction record exists for %s -> %s. Creating a new one.", sender, receiver)
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
			// It's a mutual like, so mark as a match
			newStatus = "match"
			generatedMatchID := uuid.New().String()
			matchID = &generatedMatchID

			// Update both interactions to "match" status
			log.Printf("🔥 Mutual Match Found! Updating both interactions: %s <-> %s", sender, receiver)

			// Update UserA -> UserB interaction to "match"
			err := s.UpdateInteractionStatus(ctx, receiver, sender, "match", matchID, nil)
			if err != nil {
				log.Printf("❌ Failed to update mutual match for %s -> %s: %v", receiver, sender, err)
				return err
			}
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
		log.Printf("🆕 No existing interaction found. Creating a new interaction for %s -> %s", sender, receiver)
		err := s.CreateInteraction(ctx, sender, receiver, interactionType, newStatus, matchID, message)
		if err != nil {
			log.Printf("❌ Failed to create interaction: %v", err)
			return err
		}
		log.Println("✅ New interaction successfully created.")
		return nil
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
	if err != nil {
		log.Printf("❌ Error updating interaction status: %v", err)
		return err
	}

	log.Println("✅ Interaction status successfully updated.")
	return nil
}

// ✅ GetMutualMatches using GSI instead of scan
func (s *InteractionService) GetMutualMatches(ctx context.Context, userHandle string) ([]string, error) {
	log.Printf("🔍 Fetching mutual matches for user: %s", userHandle)

	// ✅ Use GSI from models package
	indexName := models.StatusIndex

	// ✅ Query where `status = match` and `PK = USER#userHandle`
	keyCondition := "#PK = :user"
	expressionValues := map[string]types.AttributeValue{
		":user":  &types.AttributeValueMemberS{Value: "USER#" + userHandle},
		":match": &types.AttributeValueMemberS{Value: "match"},
	}
	expressionNames := map[string]string{
		"#PK":     "PK",     // ✅ User handle as partition key
		"#status": "status", // ✅ Filter only matched interactions
	}

	// ✅ Use `QueryItemsWithIndex` for efficient querying
	items, err := s.Dynamo.QueryItemsWithIndex(ctx, models.InteractionsTable, indexName, keyCondition, expressionValues, expressionNames, 100)
	if err != nil {
		log.Printf("❌ Error fetching mutual matches: %v", err)
		return nil, fmt.Errorf("failed to fetch matches: %w", err)
	}

	// ✅ Extract matched user handles
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

// ✅ GetInteractedUsers using GSI instead of Scan
func (s *InteractionService) GetInteractedUsers(ctx context.Context, userHandle string, interactionTypes []string) ([]string, error) {
	log.Printf("🔍 Fetching interacted users for: %s with types: %v", userHandle, interactionTypes)

	// ✅ Use GSI from models package
	indexName := models.InteractionTypeIndex

	// ✅ Query where `interactionType IN (...)` and `PK = USER#userHandle`
	keyCondition := "#PK = :user"
	expressionValues := map[string]types.AttributeValue{
		":user": &types.AttributeValueMemberS{Value: "USER#" + userHandle},
	}
	expressionNames := map[string]string{"#PK": "PK"}

	// ✅ Filter multiple interaction types
	if len(interactionTypes) > 0 {
		var filterExpressions []string
		for i, interactionType := range interactionTypes {
			paramName := fmt.Sprintf(":interactionType%d", i)
			expressionValues[paramName] = &types.AttributeValueMemberS{Value: interactionType}
			filterExpressions = append(filterExpressions, fmt.Sprintf("#interactionType = %s", paramName))
		}
		expressionNames["#interactionType"] = "interactionType"
		keyCondition += " AND (" + strings.Join(filterExpressions, " OR ") + ")"
	}

	// ✅ Use `QueryItemsWithIndex` for efficient querying
	items, err := s.Dynamo.QueryItemsWithIndex(ctx, models.InteractionsTable, indexName, keyCondition, expressionValues, expressionNames, 100)
	if err != nil {
		log.Printf("❌ Error querying interacted users: %v", err)
		return nil, fmt.Errorf("failed to fetch interacted users: %w", err)
	}

	// ✅ Extract interacted user handles
	users := []string{}
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

// ✅ Fetch interactions sent by the user
func (s *InteractionService) GetUserInteractions(ctx context.Context, userHandle string) ([]models.Interaction, error) {
	log.Printf("🔍 Fetching interactions SENT by user: %s", userHandle)

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

	log.Printf("✅ Found %d interactions sent by %s", len(interactions), userHandle)
	return interactions, nil
}

// GetReceivedInteractions fetches all interactions where the user is the receiver
func (s *InteractionService) GetReceivedInteractions(ctx context.Context, userHandle string) ([]models.Interaction, error) {
	log.Printf("🔍 Fetching interactions RECEIVED by user: %s", userHandle)

	// Use the new GSI (Global Secondary Index) for `receiverHandle`
	indexName := models.ReceiverHandleIndex // ✅ Ensure this index exists in DynamoDB
	keyCondition := "#receiverHandle = :receiver"
	expressionValues := map[string]types.AttributeValue{
		":receiver": &types.AttributeValueMemberS{Value: userHandle},
	}
	expressionNames := map[string]string{"#receiverHandle": "receiverHandle"}

	// ✅ Use the new QueryItemsWithIndex helper
	items, err := s.Dynamo.QueryItemsWithIndex(ctx, models.InteractionsTable, indexName, keyCondition, expressionValues, expressionNames, 100)
	if err != nil {
		log.Printf("❌ Error querying received interactions: %v", err)
		return nil, fmt.Errorf("failed to fetch received interactions: %w", err)
	}

	var interactions []models.Interaction
	err = attributevalue.UnmarshalListOfMaps(items, &interactions)
	if err != nil {
		log.Printf("❌ Error unmarshaling received interactions: %v", err)
		return nil, fmt.Errorf("failed to process received interactions: %w", err)
	}

	log.Printf("✅ Found %d interactions received by %s", len(interactions), userHandle)
	return interactions, nil
}
