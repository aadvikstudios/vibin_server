package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"vibin_server/models"

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

// HasUserLiked checks if a user has already liked another user
func (s *InteractionService) HasUserLiked(ctx context.Context, senderHandle, receiverHandle string) (bool, error) {
	keyCondition := "senderHandle = :sender AND receiverHandle = :receiver AND #type = :type"
	expressionValues := map[string]types.AttributeValue{
		":sender":   &types.AttributeValueMemberS{Value: senderHandle},
		":receiver": &types.AttributeValueMemberS{Value: receiverHandle},
		":type":     &types.AttributeValueMemberS{Value: "like"},
	}
	expressionNames := map[string]string{"#type": "type"}

	items, err := s.Dynamo.QueryItems(ctx, models.InteractionsTable, keyCondition, expressionValues, expressionNames, 1)
	if err != nil {
		return false, err
	}

	return len(items) > 0, nil
}

// IsMatch checks if two users have liked each other
func (s *InteractionService) IsMatch(ctx context.Context, senderHandle, receiverHandle string) (bool, error) {
	hasSenderLiked, err := s.HasUserLiked(ctx, senderHandle, receiverHandle)
	if err != nil || !hasSenderLiked {
		return false, err
	}

	hasReceiverLiked, err := s.HasUserLiked(ctx, receiverHandle, senderHandle)
	if err != nil || !hasReceiverLiked {
		return false, err
	}

	return true, nil
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
