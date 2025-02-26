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

func (s *InteractionService) SaveInteraction(ctx context.Context, senderHandle, receiverHandle, interactionType, message string) error {
	// ✅ Ensure Sort Key is properly formatted
	sortKey := senderHandle + "#" + interactionType

	// ✅ Create Interaction Record
	interaction := models.Interaction{
		ReceiverHandle: receiverHandle, // ✅ Partition Key (PK)
		SortKey:        sortKey,        // ✅ Sort Key (senderHandle#type)
		SenderHandle:   senderHandle,
		Type:           interactionType,
		Status:         "pending",
		CreatedAt:      time.Now().Format(time.RFC3339),
	}

	// ✅ Add message only for pings
	if interactionType == "ping" && message != "" {
		interaction.Message = &message
	}

	// ✅ Log before inserting
	log.Printf("📥 Saving interaction: %+v", interaction)

	// ✅ Save the interaction in DynamoDB
	err := s.Dynamo.PutItem(ctx, models.InteractionsTable, interaction)
	if err != nil {
		log.Printf("❌ Failed to save interaction: %v", err)
		return fmt.Errorf("failed to put item in table 'Interactions': %w", err)
	}

	log.Printf("✅ Interaction recorded: %s -> %s (%s)", senderHandle, receiverHandle, interactionType)

	// ✅ Check if it's a mutual like (Potential Match)
	if interactionType == "like" {
		isMatch, err := s.IsMatch(ctx, senderHandle, receiverHandle)
		if err != nil {
			log.Printf("⚠️ Error checking for match: %v", err)
			return nil // Don't fail if match check fails
		}
		if isMatch {
			log.Printf("🎉 It's a MATCH! %s ❤️ %s", senderHandle, receiverHandle)

			// ✅ Update the interaction status from "pending" to "match"
			err := s.UpdateInteractionStatus(ctx, senderHandle, receiverHandle, "match")
			if err != nil {
				log.Printf("⚠️ Error updating interaction status: %v", err)
			}

			err = s.UpdateInteractionStatus(ctx, receiverHandle, senderHandle, "match")
			if err != nil {
				log.Printf("⚠️ Error updating interaction status for receiver's entry: %v", err)
			}

			// ✅ Create match entry
			_, err = s.CreateMatch(ctx, senderHandle, receiverHandle)
			if err != nil {
				return err
			}
		}

	}

	return nil
}

// ✅ Corrected Query for senderHandle GSI
func (s *InteractionService) HasUserLiked(ctx context.Context, receiverHandle, senderHandle string) (bool, error) {
	log.Printf("🔍 Checking if %s has liked %s", receiverHandle, senderHandle)

	// ✅ Corrected Query Condition: Query GSI using senderHandle only
	keyCondition := "senderHandle = :sender"
	expressionValues := map[string]types.AttributeValue{
		":sender": &types.AttributeValueMemberS{Value: receiverHandle}, // ✅ Query by senderHandle
	}

	// ✅ Query senderHandle-index
	log.Printf("🔍 Querying GSI: senderHandle-index in table: %s", models.InteractionsTable)
	items, err := s.Dynamo.QueryItemsWithIndex(ctx, models.InteractionsTable, "senderHandle-index", keyCondition, expressionValues, nil, 100)
	if err != nil {
		log.Printf("❌ Error querying GSI: %v", err)
		return false, nil
	}

	// ✅ Check if the receiverHandle exists in the results
	for _, item := range items {
		var interaction models.Interaction
		err := attributevalue.UnmarshalMap(item, &interaction)
		if err != nil {
			log.Printf("❌ Error unmarshalling interaction: %v", err)
			continue
		}

		// ✅ Ensure it was a "like" interaction
		if interaction.ReceiverHandle == senderHandle && interaction.Type == "like" {
			log.Printf("✅ %s has already liked %s", receiverHandle, senderHandle)
			return true, nil
		}
	}

	log.Printf("⚠️ %s has NOT liked %s", receiverHandle, senderHandle)
	return false, nil
}

// ✅ Check if two users have mutually liked each other
func (s *InteractionService) IsMatch(ctx context.Context, senderHandle, receiverHandle string) (bool, error) {
	log.Printf("🔍 Checking match status for %s and %s", senderHandle, receiverHandle)

	// ✅ Check if receiver has already liked the sender
	hasReceiverLiked, err := s.HasUserLiked(ctx, receiverHandle, senderHandle)
	if err != nil {
		log.Printf("❌ Error checking if %s liked %s: %v", receiverHandle, senderHandle, err)
		return false, nil
	}

	// ✅ If receiver has liked sender, it's a match!
	if hasReceiverLiked {
		log.Printf("🎉 Match confirmed: %s ❤️ %s", senderHandle, receiverHandle)
		return true, nil
	}

	log.Printf("⚠️ No match yet for %s and %s", senderHandle, receiverHandle)
	return false, nil
}

// CreateMatch - Stores a match in the Matches table and returns the MatchID
func (s *InteractionService) CreateMatch(ctx context.Context, user1, user2 string) (string, error) {
	matchID := uuid.New().String()
	createdAt := time.Now().Format(time.RFC3339)

	match := models.Match{
		MatchID:     matchID,
		User1Handle: user1,
		User2Handle: user2,
		Status:      "active",
		CreatedAt:   createdAt,
	}

	// ✅ Save match in DynamoDB
	err := s.Dynamo.PutItem(ctx, models.MatchesTable, match)
	if err != nil {
		log.Printf("❌ Failed to create match: %v", err)
		return "", fmt.Errorf("failed to create match: %w", err)
	}

	log.Printf("🎉 Match created: %s ❤️ %s", user1, user2)
	return matchID, nil
}

// SendInitialMessage - Sends a default welcome message to start the chat
func (s *InteractionService) SendInitialMessage(ctx context.Context, matchID, senderHandle, receiverHandle string) error {
	messageID := uuid.New().String()
	createdAt := time.Now().Format(time.RFC3339)

	message := models.Message{
		MatchID:   matchID,
		MessageID: messageID,
		SenderID:  senderHandle,
		Content:   "Hey! You both matched! 🎉 Start a conversation now!",
		IsUnread:  true,
		Liked:     false,
		CreatedAt: createdAt,
	}

	// ✅ Save message in DynamoDB
	err := s.Dynamo.PutItem(ctx, models.MessagesTable, message)
	if err != nil {
		log.Printf("❌ Failed to send initial message: %v", err)
		return fmt.Errorf("failed to send initial message: %w", err)
	}

	log.Printf("📩 Initial message sent for Match %s: %s", matchID, message.Content)
	return nil
}

// UpdateInteractionStatus - Update the status of an interaction (like, ping, etc.)
func (s *InteractionService) UpdateInteractionStatus(ctx context.Context, senderHandle, receiverHandle, newStatus string) error {
	log.Printf("🔄 Updating interaction status to '%s' for %s -> %s", newStatus, senderHandle, receiverHandle)

	// Define Key (PK and SK)
	key := map[string]types.AttributeValue{
		"receiverHandle": &types.AttributeValueMemberS{Value: receiverHandle},
		"sk":             &types.AttributeValueMemberS{Value: senderHandle + "#ping"}, // ✅ Ensure it updates a ping interaction
	}

	// Define Update Expression
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
