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

// ✅ SaveInteraction now ensures the correct keys are used for indexing
func (s *InteractionService) SaveInteraction(ctx context.Context, senderHandle, receiverHandle, interactionType string) error {
	createdAt := time.Now().Format(time.RFC3339)

	interaction := models.Interaction{
		ReceiverHandle: receiverHandle,
		SenderHandle:   senderHandle,
		Type:           interactionType,
		Status:         "pending",
		CreatedAt:      createdAt,
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

// ✅ Optimized `HasUserLiked` to use `senderHandle-index`
func (s *InteractionService) HasUserLiked(ctx context.Context, senderHandle, receiverHandle string) (bool, error) {
	log.Printf("🔍 Checking if %s has liked %s", senderHandle, receiverHandle)

	// ✅ Query using GSI (`senderHandle-index`) to fetch all likes sent by `senderHandle`
	keyCondition := "senderHandle = :sender AND #type = :type"
	expressionValues := map[string]types.AttributeValue{
		":sender": &types.AttributeValueMemberS{Value: senderHandle},
		":type":   &types.AttributeValueMemberS{Value: "like"},
	}
	expressionNames := map[string]string{"#type": "type"}

	// ✅ Query DynamoDB using the GSI
	items, err := s.Dynamo.QueryItemsWithIndex(ctx, models.InteractionsTable, models.SenderHandleIndex, keyCondition, expressionValues, expressionNames, 100)
	if err != nil {
		log.Printf("❌ Error querying likes from senderHandle-index for %s: %v", senderHandle, err)
		return false, nil
	}

	// ✅ Check if receiverHandle exists in the results
	for _, item := range items {
		var interaction models.Interaction
		err := attributevalue.UnmarshalMap(item, &interaction)
		if err != nil {
			log.Printf("❌ Error unmarshalling interaction: %v", err)
			continue
		}
		if interaction.ReceiverHandle == receiverHandle {
			log.Printf("✅ %s has already liked %s", senderHandle, receiverHandle)
			return true, nil
		}
	}

	log.Printf("⚠️ %s has NOT liked %s", senderHandle, receiverHandle)
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
