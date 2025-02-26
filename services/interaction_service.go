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

// ✅ Updated SaveInteraction to accept a message parameter
func (s *InteractionService) SaveInteraction(ctx context.Context, senderHandle, receiverHandle, interactionType, message string) error {
	createdAt := time.Now().Format(time.RFC3339)

	interaction := models.Interaction{
		ReceiverHandle: receiverHandle,
		SenderHandle:   senderHandle,
		Type:           interactionType,
		Status:         "pending",
		CreatedAt:      createdAt,
	}

	// ✅ Only add message if it's a "ping"
	if interactionType == "ping" && message != "" {
		interaction.Message = message
	}

	// ✅ Save the interaction in DynamoDB
	err := s.Dynamo.PutItem(ctx, models.InteractionsTable, interaction)
	if err != nil {
		log.Printf("❌ Failed to save interaction: %v", err)
		return fmt.Errorf("failed to save interaction: %w", err)
	}

	log.Printf("✅ Interaction recorded: %s -> %s (%s)", senderHandle, receiverHandle, interactionType)
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

// ✅ Optimized `IsMatch` function to check if both users liked each other
func (s *InteractionService) IsMatch(ctx context.Context, senderHandle, receiverHandle string) (bool, error) {
	log.Printf("🔍 Checking match status for %s and %s", senderHandle, receiverHandle)

	// ✅ Check if receiver has liked the sender
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

// ✅ CreateMatch function creates a match when both users like each other
func (s *InteractionService) CreateMatch(ctx context.Context, user1, user2 string) error {
	matchID := uuid.New().String()
	createdAt := time.Now().Format(time.RFC3339)

	match := models.Match{
		MatchID:     matchID,
		User1Handle: user1,
		User2Handle: user2,
		Status:      "active",
		CreatedAt:   createdAt,
	}

	err := s.Dynamo.PutItem(ctx, models.MatchesTable, match)
	if err != nil {
		return fmt.Errorf("failed to create match: %w", err)
	}

	log.Printf("🎉 Match created: %s ❤️ %s", user1, user2)
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

	keyCondition := "receiverHandle = :receiver"
	expressionValues := map[string]types.AttributeValue{
		":receiver": &types.AttributeValueMemberS{Value: receiverHandle},
	}

	items, err := s.Dynamo.QueryItems(ctx, models.InteractionsTable, keyCondition, expressionValues, nil, 100)
	if err != nil {
		log.Printf("❌ Error querying interactions: %v", err)
		return nil, fmt.Errorf("failed to fetch interactions: %w", err)
	}

	var enrichedInteractions []models.InteractionWithProfile

	for _, item := range items {
		var interaction models.Interaction
		err := attributevalue.UnmarshalMap(item, &interaction)
		if err != nil {
			log.Printf("❌ Error unmarshalling interaction: %v", err)
			continue
		}

		// ✅ Corrected: Use a map for key
		userProfileKey := map[string]types.AttributeValue{
			"userhandle": &types.AttributeValueMemberS{Value: interaction.SenderHandle},
		}

		// ✅ Fetch user profile using the corrected function call
		userProfile, err := s.Dynamo.GetItem(ctx, models.UserProfilesTable, userProfileKey)
		if err != nil {
			log.Printf("⚠️ Warning: Failed to fetch user profile for %s: %v", interaction.SenderHandle, err)
			userProfile = map[string]types.AttributeValue{} // Empty profile
		}

		// ✅ Convert profile data from DynamoDB to struct
		var userProfileData models.UserProfile
		err = attributevalue.UnmarshalMap(userProfile, &userProfileData)
		if err != nil {
			log.Printf("⚠️ Warning: Failed to parse user profile data: %v", err)
			continue
		}

		// ✅ Merge into `InteractionWithProfile` struct
		combinedData := models.InteractionWithProfile{
			ReceiverHandle: interaction.ReceiverHandle,
			SenderHandle:   interaction.SenderHandle,
			Type:           interaction.Type,
			Message:        interaction.Message,
			Status:         interaction.Status,
			CreatedAt:      interaction.CreatedAt,

			// Profile fields
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
