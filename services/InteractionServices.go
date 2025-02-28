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
	Dynamo      *DynamoService
	ChatService *ChatService
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

func (s *InteractionService) CreateOrUpdateInteraction(
	ctx context.Context, sender, receiver, interactionType, action string, message *string) error {
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

		// ✅ Check if it's a mutual match
		isMatch, err := s.CheckMutualMatch(ctx, sender, receiver)
		if err != nil {
			return err
		}

		// ✅ If mutual match, handle it
		if isMatch {
			newStatus = "match"
			matchID, err = s.HandleMutualMatch(ctx, sender, receiver)
			if err != nil {
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
func (s *InteractionService) HandlePingApproval(ctx context.Context, sender, receiver string) error {
	log.Printf("✅ Handling Ping Approval: %s -> %s", sender, receiver)

	// ✅ Update status to "match"
	matchID := uuid.New().String()
	err := s.UpdateInteractionStatus(ctx, sender, receiver, "match", &matchID, nil)
	if err != nil {
		log.Printf("❌ Failed to approve ping: %v", err)
		return err
	}

	// ✅ Also update reverse interaction (Receiver -> Sender)
	err = s.UpdateInteractionStatus(ctx, receiver, sender, "match", &matchID, nil)
	if err != nil {
		log.Printf("⚠️ Failed to update reverse ping status: %v", err)
	}

	// ✅ Send an initial message (with original ping content)
	err = s.CreateInitialMessage(ctx, sender, receiver, matchID, true)
	if err != nil {
		log.Printf("⚠️ Failed to send initial message: %v", err)
	}

	log.Printf("✅ Ping Approved: %s <-> %s", sender, receiver)
	return nil
}

func (s *InteractionService) HandlePingDecline(ctx context.Context, sender, receiver string) error {
	log.Printf("🚫 Handling Ping Decline: %s -> %s", sender, receiver)

	// ✅ Update status to "declined"
	err := s.UpdateInteractionStatus(ctx, sender, receiver, "declined", nil, nil)
	if err != nil {
		log.Printf("❌ Failed to decline ping: %v", err)
		return err
	}

	// ✅ Also update reverse interaction (Receiver -> Sender)
	err = s.UpdateInteractionStatus(ctx, receiver, sender, "declined", nil, nil)
	if err != nil {
		log.Printf("⚠️ Failed to update reverse ping status: %v", err)
	}

	log.Printf("✅ Ping Declined: %s -> %s", sender, receiver)
	return nil
}

func (s *InteractionService) CheckMutualMatch(ctx context.Context, sender, receiver string) (bool, error) {
	log.Printf("🔍 Checking for mutual match: %s <-> %s", sender, receiver)

	// Fetch existing interaction (if any) where receiver liked sender
	mutualLike, err := s.GetInteraction(ctx, receiver, sender)
	if err != nil {
		log.Printf("❌ Error fetching interaction for mutual match check: %v", err)
		return false, err
	}

	// ✅ If the receiver also liked the sender, it's a mutual match
	if mutualLike != nil && mutualLike.Status == "pending" {
		log.Printf("🔥 Mutual Match Found! %s <-> %s", sender, receiver)
		return true, nil
	}

	// ❌ No mutual match
	return false, nil
}
func (s *InteractionService) HandleMutualMatch(ctx context.Context, sender, receiver string) (*string, error) {
	log.Printf("🔄 Handling mutual match update for: %s <-> %s", sender, receiver)

	// Generate a match ID
	matchID := uuid.New().String()

	// ✅ Update UserB -> UserA interaction to "match"
	err := s.UpdateInteractionStatus(ctx, receiver, sender, "match", &matchID, nil)
	if err != nil {
		log.Printf("❌ Failed to update mutual match for %s -> %s: %v", receiver, sender, err)
		return nil, err
	}

	// ✅ Create an initial message (default congratulatory message)
	err = s.CreateInitialMessage(ctx, sender, receiver, matchID, false)
	if err != nil {
		log.Printf("⚠️ Failed to send initial message for match %s: %v", matchID, err)
	}

	return &matchID, nil
}

func (s *InteractionService) CreateInitialMessage(ctx context.Context, sender, receiver, matchID string, isPing bool) error {
	log.Printf("💬 Creating initial message for matchId: %s between %s & %s", matchID, sender, receiver)

	// Determine message content and sender
	var content string
	var originalSender string

	if isPing {
		// ✅ Fetch the original ping interaction to get the message content
		originalInteraction, err := s.GetInteraction(ctx, sender, receiver)
		if err != nil {
			log.Printf("❌ Failed to fetch original ping interaction: %v", err)
			return err
		}

		if originalInteraction == nil || originalInteraction.Message == nil {
			log.Printf("⚠️ No original ping message found, using default content")
			content = "Hey! I sent you a ping. Let's connect! 😊"
		} else {
			content = *originalInteraction.Message // ✅ Use original ping message
		}

		originalSender = sender // ✅ Keep the original sender
	} else {
		// ✅ Default message for mutual like
		content = "Congratulations! You both liked each other. Say hello! 👋"
		originalSender = sender
	}

	// ✅ Define the first message
	initialMessage := models.Message{
		MatchID:   matchID,
		MessageID: uuid.New().String(),
		SenderID:  originalSender, // ✅ Keep the original sender
		Content:   content,
		CreatedAt: time.Now().Format(time.RFC3339), // Store timestamp
		Liked:     false,                           // Default to false
	}

	// ✅ Set IsUnread using helper method
	initialMessage.SetIsUnread(true)

	// ✅ Send message using ChatService
	err := s.ChatService.SendMessage(ctx, initialMessage)
	if err != nil {
		log.Printf("❌ Failed to send initial message: %v", err)
		return err
	}

	log.Printf("✅ Initial message sent successfully for matchId: %s", matchID)
	return nil
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
func (s *InteractionService) GetMutualMatches(ctx context.Context, userHandle string) ([]string, error) {
	log.Printf("🔍 Fetching mutual matches for user: %s", userHandle)

	// ✅ Use the correct index with `PK = USER#userHandle` and `SK = status`
	indexName := models.StatusIndex
	keyCondition := "#PK = :user AND #status = :matchStatus"

	// ✅ Ensure `PK` includes "USER#"
	expressionValues := map[string]types.AttributeValue{
		":user":        &types.AttributeValueMemberS{Value: "USER#" + userHandle}, // ✅ Correct PK format
		":matchStatus": &types.AttributeValueMemberS{Value: "match"},
	}

	expressionNames := map[string]string{
		"#PK":     "PK", // ✅ Match the GSI PK (which now follows table PK format)
		"#status": "status",
	}

	// ✅ Query using the optimized index
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
			log.Printf("⚠️ Skipping item due to unmarshalling error: %v", err)
			continue
		}

		// ✅ Ensure the receiverHandle is valid before appending
		if interaction.ReceiverHandle != "" {
			matches = append(matches, interaction.ReceiverHandle)
		} else {
			log.Printf("⚠️ Skipping item with empty receiverHandle: %+v", interaction)
		}
	}

	log.Printf("✅ Found %d mutual matches for %s", len(matches), userHandle)
	return matches, nil
}

func (s *InteractionService) GetInteractedUsers(ctx context.Context, userHandle string, interactionTypes []string) ([]string, error) {
	log.Printf("🔍 Fetching interacted users for: %s with types: %v", userHandle, interactionTypes)

	// ✅ Ensure the correct GSI name is used
	indexName := models.InteractionTypeIndex

	// ✅ Use 'interactionType' in KeyConditionExpression (Not FilterExpression)
	var keyConditions []string
	expressionValues := map[string]types.AttributeValue{
		":userHandle": &types.AttributeValueMemberS{Value: "USER#" + userHandle},
	}
	expressionNames := map[string]string{
		"#PK":              "PK",
		"#interactionType": "interactionType",
	}

	// ✅ KeyConditionExpression supports "IN" only if it's a Sort Key
	if len(interactionTypes) == 1 {
		expressionValues[":interactionType"] = &types.AttributeValueMemberS{Value: interactionTypes[0]}
		keyConditions = append(keyConditions, "#PK = :userHandle AND #interactionType = :interactionType")
	} else {
		// ✅ Use "OR" alternative: Query multiple times if needed
		var interactedUsers []string
		for _, interactionType := range interactionTypes {
			log.Printf("🔄 Querying for interaction type: %s", interactionType)

			tempExpressionValues := map[string]types.AttributeValue{
				":userHandle":      expressionValues[":userHandle"],
				":interactionType": &types.AttributeValueMemberS{Value: interactionType},
			}

			items, err := s.Dynamo.QueryItemsWithIndex(
				ctx, models.InteractionsTable, indexName,
				"#PK = :userHandle AND #interactionType = :interactionType",
				tempExpressionValues, expressionNames, 50,
			)
			if err != nil {
				log.Printf("❌ Error querying interactionType '%s': %v", interactionType, err)
				continue // Skip this type but continue others
			}

			for _, item := range items {
				var interaction models.Interaction
				if err := attributevalue.UnmarshalMap(item, &interaction); err == nil {
					interactedUsers = append(interactedUsers, interaction.ReceiverHandle)
				}
			}
		}
		log.Printf("✅ Total Interacted Users Found: %d", len(interactedUsers))
		return interactedUsers, nil
	}

	// ✅ Query with the correct key conditions
	log.Printf("🔍 Querying GSI '%s' with condition: %s", indexName, keyConditions[0])
	items, err := s.Dynamo.QueryItemsWithIndex(ctx, models.InteractionsTable, indexName, keyConditions[0], expressionValues, expressionNames, 50)
	if err != nil {
		log.Printf("❌ Error querying interacted users: %v", err)
		return nil, fmt.Errorf("failed to fetch interacted users: %w", err)
	}

	// ✅ Extract interacted user handles
	users := []string{}
	for _, item := range items {
		var interaction models.Interaction
		if err := attributevalue.UnmarshalMap(item, &interaction); err == nil {
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
