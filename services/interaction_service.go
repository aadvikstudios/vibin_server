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

type InteractionService struct {
	Dynamo *DynamoService
}

// SaveInteraction handles like and ping interactions
func (s *InteractionService) SaveInteraction(ctx context.Context, senderHandle, receiverHandle, interactionType, message string) error {
	// Define the sort key
	sortKey := senderHandle + "#" + interactionType

	// Create an interaction record
	interaction := models.Interaction{
		ReceiverHandle: receiverHandle,
		SortKey:        sortKey,
		SenderHandle:   senderHandle,
		Type:           interactionType,
		Status:         "pending",
		CreatedAt:      time.Now().Format(time.RFC3339),
	}

	// Store message only if it's a ping
	if interactionType == "ping" && message != "" {
		interaction.Message = &message
	}

	log.Printf("📥 Saving interaction: %+v", interaction)

	// Save the interaction
	if err := s.Dynamo.PutItem(ctx, models.InteractionsTable, interaction); err != nil {
		log.Printf("❌ Failed to save interaction: %v", err)
		return fmt.Errorf("failed to save interaction: %w", err)
	}

	log.Printf("✅ Interaction recorded: %s -> %s (%s)", senderHandle, receiverHandle, interactionType)

	// Handle match scenario
	if interactionType == "like" {
		isMatch, err := s.IsMatch(ctx, senderHandle, receiverHandle)
		if err != nil {
			log.Printf("⚠️ Error checking for match: %v", err)
			return nil
		}
		if isMatch {
			return s.HandleMatch(ctx, senderHandle, receiverHandle, "")
		}
	}

	return nil
}

// HandleMatch updates interaction statuses, creates a match, and inserts a message
func (s *InteractionService) HandleMatch(ctx context.Context, user1, user2, message string) error {
	log.Printf("🎉 Creating match between %s ❤️ %s", user1, user2)

	// Update interaction statuses
	if err := s.UpdateInteractionStatus(ctx, user1, user2, "match"); err != nil {
		log.Printf("⚠️ Error updating status for %s -> %s: %v", user1, user2, err)
	}
	if err := s.UpdateInteractionStatus(ctx, user2, user1, "match"); err != nil {
		log.Printf("⚠️ Error updating status for %s -> %s: %v", user2, user1, err)
	}

	// Create match entry
	matchID, err := s.CreateMatch(ctx, user1, user2)
	if err != nil {
		return err
	}

	// Insert initial message (empty for mutual like, contains message for ping approval)
	if err := s.SendInitialMessage(ctx, matchID, user1, user2, message); err != nil {
		return err
	}

	return nil
}

// UpdateInteractionStatus ensures an existing record is updated instead of inserting a new one
func (s *InteractionService) UpdateInteractionStatus(ctx context.Context, senderHandle, receiverHandle, newStatus string) error {
	log.Printf("🔄 Updating interaction status to '%s' for %s -> %s", newStatus, senderHandle, receiverHandle)

	// Define key for updating the existing interaction record
	key := map[string]types.AttributeValue{
		"receiverHandle": &types.AttributeValueMemberS{Value: receiverHandle},
		"sk":             &types.AttributeValueMemberS{Value: senderHandle + "#like"},
	}

	// Define the update expression
	updateExpression := "SET #status = :status"
	expressionValues := map[string]types.AttributeValue{
		":status": &types.AttributeValueMemberS{Value: newStatus},
	}
	expressionNames := map[string]string{
		"#status": "status",
	}

	// Perform the update
	_, err := s.Dynamo.UpdateItem(ctx, models.InteractionsTable, updateExpression, key, expressionValues, expressionNames)
	if err != nil {
		log.Printf("❌ Error updating interaction status: %v", err)
		return fmt.Errorf("failed to update interaction status: %w", err)
	}

	log.Printf("✅ Successfully updated interaction status to '%s' for %s -> %s", newStatus, senderHandle, receiverHandle)
	return nil
}

// GetPingMessage retrieves the original message from the ping interaction
func (s *InteractionService) GetPingMessage(ctx context.Context, senderHandle, receiverHandle string) (string, error) {
	log.Printf("🔍 Fetching ping message for %s -> %s", senderHandle, receiverHandle)

	key := map[string]types.AttributeValue{
		"receiverHandle": &types.AttributeValueMemberS{Value: receiverHandle},
		"sk":             &types.AttributeValueMemberS{Value: senderHandle + "#ping"},
	}

	item, err := s.Dynamo.GetItem(ctx, models.InteractionsTable, key)
	if err != nil {
		log.Printf("❌ Error fetching ping message: %v", err)
		return "", err
	}

	var interaction models.Interaction
	if err := attributevalue.UnmarshalMap(item, &interaction); err != nil {
		log.Printf("❌ Error unmarshalling interaction: %v", err)
		return "", err
	}

	if interaction.Message != nil {
		return *interaction.Message, nil
	}

	return "", nil // No message found
}

// IsMatch checks if two users have liked each other
func (s *InteractionService) IsMatch(ctx context.Context, senderHandle, receiverHandle string) (bool, error) {
	log.Printf("🔍 Checking match status for %s and %s", senderHandle, receiverHandle)

	hasReceiverLiked, err := s.HasUserLiked(ctx, receiverHandle, senderHandle)
	if err != nil {
		log.Printf("❌ Error checking if %s liked %s: %v", receiverHandle, senderHandle, err)
		return false, nil
	}

	if hasReceiverLiked {
		log.Printf("🎉 Match confirmed: %s ❤️ %s", senderHandle, receiverHandle)
		return true, nil
	}

	log.Printf("⚠️ No match yet for %s and %s", senderHandle, receiverHandle)
	return false, nil
}

// HasUserLiked checks if a user has already liked another user
func (s *InteractionService) HasUserLiked(ctx context.Context, receiverHandle, senderHandle string) (bool, error) {
	log.Printf("🔍 Checking if %s has liked %s", receiverHandle, senderHandle)

	keyCondition := "senderHandle = :sender"
	expressionValues := map[string]types.AttributeValue{
		":sender": &types.AttributeValueMemberS{Value: receiverHandle},
	}

	items, err := s.Dynamo.QueryItemsWithIndex(ctx, models.InteractionsTable, "senderHandle-index", keyCondition, expressionValues, nil, 100)
	if err != nil {
		log.Printf("❌ Error querying interactions: %v", err)
		return false, nil
	}

	for _, item := range items {
		var interaction models.Interaction
		err := attributevalue.UnmarshalMap(item, &interaction)
		if err != nil {
			continue
		}

		if interaction.ReceiverHandle == senderHandle && interaction.Type == "like" {
			return true, nil
		}
	}

	return false, nil
}

// CreateMatch inserts a match record in the Matches table
func (s *InteractionService) CreateMatch(ctx context.Context, user1, user2 string) (string, error) {
	matchID := uuid.New().String()

	match := models.Match{
		MatchID:     matchID,
		User1Handle: user1,
		User2Handle: user2,
		Status:      "active",
		CreatedAt:   time.Now().Format(time.RFC3339),
	}

	if err := s.Dynamo.PutItem(ctx, models.MatchesTable, match); err != nil {
		log.Printf("❌ Failed to create match: %v", err)
		return "", fmt.Errorf("failed to create match: %w", err)
	}

	return matchID, nil
}

func (s *InteractionService) SendInitialMessage(ctx context.Context, matchID, senderHandle, receiverHandle, message string) error {
	messageID := uuid.New().String()
	createdAt := time.Now().Format(time.RFC3339)

	// If no message was provided (mutual like case), use a default message
	if message == "" {
		message = "Hey! You both matched! 🎉 Start a conversation now!"
	}

	newMessage := models.Message{
		MatchID:   matchID,
		MessageID: messageID,
		SenderID:  senderHandle,
		Content:   message,
		IsUnread:  "true", // ✅ Store as string "true"
		Liked:     false,
		CreatedAt: createdAt,
	}

	if err := s.Dynamo.PutItem(ctx, models.MessagesTable, newMessage); err != nil {
		return fmt.Errorf("failed to send initial message: %w", err)
	}

	log.Printf("📩 Initial message sent for Match %s: %s", matchID, message)
	return nil
}

// ✅ GetLikedOrDislikedUsers now correctly fetches interactions using GSI
func (s *InteractionService) GetLikedOrDislikedUsers(ctx context.Context, senderHandle string) (map[string]bool, error) {
	log.Printf("🔍 Fetching interactions for %s", senderHandle)

	// ✅ Query interactions where senderHandle = senderHandle
	keyCondition := "senderHandle = :sender"
	expressionValues := map[string]types.AttributeValue{
		":sender": &types.AttributeValueMemberS{Value: senderHandle},
	}

	// ✅ Use GSI (`senderHandle-index`) for efficient querying
	items, err := s.Dynamo.QueryItemsWithIndex(ctx, models.InteractionsTable, "senderHandle-index", keyCondition, expressionValues, nil, 100)
	if err != nil {
		log.Printf("❌ Error querying interactions: %v", err)
		return nil, fmt.Errorf("failed to fetch interactions: %w", err)
	}

	likedDislikedUsers := make(map[string]bool)
	for _, item := range items {
		var interaction models.Interaction
		err := attributevalue.UnmarshalMap(item, &interaction)
		if err != nil {
			log.Printf("❌ Error unmarshalling interaction: %v", err)
			continue
		}
		likedDislikedUsers[interaction.ReceiverHandle] = true
	}

	log.Printf("✅ Found %d interactions for %s", len(likedDislikedUsers), senderHandle)
	return likedDislikedUsers, nil
}

func (s *InteractionService) GetInteractionsByReceiverHandle(ctx context.Context, receiverHandle string) ([]models.InteractionWithProfile, error) {
	log.Printf("🔍 Querying interactions where receiverHandle = %s", receiverHandle)

	// ✅ Fetch interactions for the given receiverHandle
	interactions, err := s.GetInteractionsForReceiver(ctx, receiverHandle)
	if err != nil {
		log.Printf("❌ Error fetching interactions: %v", err)
		return nil, err
	}

	log.Printf("✅ Found %d interactions for receiverHandle: %s", len(interactions), receiverHandle)

	// ✅ Enrich interactions with user profile data
	return s.EnrichInteractionsWithProfiles(ctx, interactions)
}

// ✅ Fetch interactions from DynamoDB
func (s *InteractionService) GetInteractionsForReceiver(ctx context.Context, receiverHandle string) ([]models.Interaction, error) {
	keyCondition := "receiverHandle = :receiver"
	expressionValues := map[string]types.AttributeValue{
		":receiver": &types.AttributeValueMemberS{Value: receiverHandle},
	}

	items, err := s.Dynamo.QueryItems(ctx, models.InteractionsTable, keyCondition, expressionValues, nil, 100)
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

// ✅ Fetch user profiles for interactions and merge them
func (s *InteractionService) EnrichInteractionsWithProfiles(ctx context.Context, interactions []models.Interaction) ([]models.InteractionWithProfile, error) {
	var enrichedInteractions []models.InteractionWithProfile

	for _, interaction := range interactions {
		// Fetch sender's profile from UserProfiles table
		userProfileKey := map[string]types.AttributeValue{
			"userhandle": &types.AttributeValueMemberS{Value: interaction.SenderHandle},
		}

		userProfileItem, err := s.Dynamo.GetItem(ctx, models.UserProfilesTable, userProfileKey)
		if err != nil {
			log.Printf("⚠️ Warning: Failed to fetch profile for %s: %v", interaction.SenderHandle, err)
			userProfileItem = map[string]types.AttributeValue{} // Empty profile
		}

		// Convert profile data from DynamoDB to struct
		var userProfileData models.UserProfile
		err = attributevalue.UnmarshalMap(userProfileItem, &userProfileData)
		if err != nil {
			log.Printf("⚠️ Warning: Failed to parse profile data for %s: %v", interaction.SenderHandle, err)
			continue
		}

		// ✅ Merge interaction and profile data
		combinedData := models.InteractionWithProfile{
			ReceiverHandle: interaction.ReceiverHandle,
			SenderHandle:   interaction.SenderHandle,
			Type:           interaction.Type,
			Message: func() string { // ✅ Safely handle *string
				if interaction.Message != nil {
					return *interaction.Message
				}
				return ""
			}(),
			Status:    interaction.Status,
			CreatedAt: interaction.CreatedAt,

			// Profile Fields
			Name:          userProfileData.Name,
			UserName:      userProfileData.UserName,
			Age:           userProfileData.Age,
			Gender:        userProfileData.Gender,
			Orientation:   userProfileData.Orientation,
			LookingFor:    userProfileData.LookingFor,
			Photos:        userProfileData.Photos,
			Bio:           userProfileData.Bio,
			Interests:     userProfileData.Interests,
			Questionnaire: userProfileData.Questionnaire,
		}

		enrichedInteractions = append(enrichedInteractions, combinedData)
	}

	return enrichedInteractions, nil
}
