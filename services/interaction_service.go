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

// SaveInteraction saves a like or dislike interaction
func (s *InteractionService) SaveInteraction(ctx context.Context, senderHandle, receiverHandle, interactionType string) error {
	interactionID := uuid.New().String()
	createdAt := time.Now().Format(time.RFC3339)

	interaction := models.Interaction{
		InteractionID:  interactionID,
		SenderHandle:   senderHandle,
		ReceiverHandle: receiverHandle,
		Type:           interactionType,
		Status:         "pending",
		CreatedAt:      createdAt,
	}

	// Save the interaction in DynamoDB
	err := s.Dynamo.PutItem(ctx, models.InteractionsTable, interaction)
	if err != nil {
		log.Printf("❌ Failed to save interaction: %v", err)
		return fmt.Errorf("failed to save interaction: %w", err)
	}

	log.Printf("✅ Interaction recorded: %s -> %s (%s)", senderHandle, receiverHandle, interactionType)
	return nil
}

// HasUserLiked checks if the given receiver has liked the sender (i.e., mutual match check)
func (s *InteractionService) HasUserLiked(ctx context.Context, receiverHandle, senderHandle string) (bool, error) {
	log.Printf("🔍 Checking if %s has liked %s", receiverHandle, senderHandle)

	// Query DynamoDB using receiverHandle as partition key
	keyCondition := "receiverHandle = :receiver AND #type = :type"
	expressionValues := map[string]types.AttributeValue{
		":receiver": &types.AttributeValueMemberS{Value: receiverHandle},
		":type":     &types.AttributeValueMemberS{Value: "like"},
	}
	expressionNames := map[string]string{"#type": "type"}

	// Query interactions table
	items, err := s.Dynamo.QueryItems(ctx, models.InteractionsTable, keyCondition, expressionValues, expressionNames, 1)
	if err != nil {
		log.Printf("❌ Error querying likes for %s: %v", receiverHandle, err)
		return false, nil
	}

	// Check if senderHandle is in the result (has liked the sender)
	for _, item := range items {
		var interaction models.Interaction
		err := attributevalue.UnmarshalMap(item, &interaction)
		if err != nil {
			log.Printf("❌ Error unmarshalling interaction: %v", err)
			continue
		}
		if interaction.SenderHandle == senderHandle {
			log.Printf("✅ %s has already liked %s", receiverHandle, senderHandle)
			return true, nil
		}
	}

	log.Printf("⚠️ %s has NOT liked %s", receiverHandle, senderHandle)
	return false, nil
}

// IsMatch checks if two users have liked each other
func (s *InteractionService) IsMatch(ctx context.Context, senderHandle, receiverHandle string) (bool, error) {
	log.Printf("🔍 Checking match status for %s and %s", senderHandle, receiverHandle)

	// Check if receiver has liked the sender
	hasReceiverLiked, err := s.HasUserLiked(ctx, receiverHandle, senderHandle)
	if err != nil {
		log.Printf("❌ Error checking if %s liked %s: %v", receiverHandle, senderHandle, err)
		return false, nil
	}

	// If receiver has liked sender, it's a match!
	if hasReceiverLiked {
		log.Printf("🎉 Match confirmed: %s ❤️ %s", senderHandle, receiverHandle)
		return true, nil
	}

	log.Printf("⚠️ No match yet for %s and %s", senderHandle, receiverHandle)
	return false, nil
}

// CreateMatch creates a match if two users have liked each other
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

// ✅ GetLikedOrDislikedUsers returns a map of users who were liked/disliked
func (s *InteractionService) GetLikedOrDislikedUsers(ctx context.Context, senderHandle string) (map[string]bool, error) {
	log.Printf("🔍 Fetching interactions for %s", senderHandle)

	// Query interactions where senderHandle = senderHandle
	keyCondition := "senderHandle = :sender"
	expressionValues := map[string]types.AttributeValue{
		":sender": &types.AttributeValueMemberS{Value: senderHandle},
	}

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
