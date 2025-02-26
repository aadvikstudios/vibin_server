package services

import (
	"context"
	"fmt"
	"log"
	"vibin_server/models"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// MatchService struct
type MatchService struct {
	Dynamo *DynamoService
}

// GetMatchesByUserHandle fetches matches where userHandle is either user1Handle or user2Handle
func (s *MatchService) GetMatchesByUserHandle(ctx context.Context, userHandle string) ([]models.Match, error) {
	var matches []models.Match

	// ✅ Query user1Handle-index
	log.Printf("🔍 Querying matches where userHandle is user1Handle: %s", userHandle)
	user1Condition := "user1Handle = :userHandle"
	expressionValues := map[string]types.AttributeValue{
		":userHandle": &types.AttributeValueMemberS{Value: userHandle},
	}

	user1Matches, err := s.Dynamo.QueryItemsWithIndex(ctx, models.MatchesTable, "user1Handle-index", user1Condition, expressionValues, nil, 100)
	if err != nil {
		log.Printf("❌ Error querying user1Handle-index: %v", err)
		return nil, fmt.Errorf("failed to fetch matches: %w", err)
	}

	// ✅ Unmarshal results
	for _, item := range user1Matches {
		var match models.Match
		if err := attributevalue.UnmarshalMap(item, &match); err != nil {
			log.Printf("❌ Error unmarshalling match from user1Handle-index: %v", err)
			continue
		}
		matches = append(matches, match)
	}

	// ✅ Query user2Handle-index
	log.Printf("🔍 Querying matches where userHandle is user2Handle: %s", userHandle)
	user2Condition := "user2Handle = :userHandle"

	user2Matches, err := s.Dynamo.QueryItemsWithIndex(ctx, models.MatchesTable, "user2Handle-index", user2Condition, expressionValues, nil, 100)
	if err != nil {
		log.Printf("❌ Error querying user2Handle-index: %v", err)
		return nil, fmt.Errorf("failed to fetch matches: %w", err)
	}

	// ✅ Unmarshal results
	for _, item := range user2Matches {
		var match models.Match
		if err := attributevalue.UnmarshalMap(item, &match); err != nil {
			log.Printf("❌ Error unmarshalling match from user2Handle-index: %v", err)
			continue
		}
		matches = append(matches, match)
	}

	log.Printf("✅ Found %d matches for userHandle: %s", len(matches), userHandle)
	return matches, nil
}
