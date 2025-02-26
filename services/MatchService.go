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

// GetMatchesByUserHandle fetches matches and enriches them with the matched user's profile
func (s *MatchService) GetMatchesByUserHandle(ctx context.Context, userHandle string) ([]models.MatchWithProfile, error) {
	// ✅ Fetch matches as []models.Match
	matches, err := s.FetchMatches(ctx, userHandle)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch matches: %w", err)
	}

	// ✅ Enrich matches with the matched user's profile
	enrichedMatches, err := s.EnrichMatchesWithProfiles(ctx, userHandle, matches)
	if err != nil {
		return nil, fmt.Errorf("failed to enrich matches with profiles: %w", err)
	}

	return enrichedMatches, nil
}

// FetchMatches queries the Matches table using both indexes
func (s *MatchService) FetchMatches(ctx context.Context, userHandle string) ([]models.Match, error) {
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
		return nil, err
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
		return nil, err
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

// EnrichMatchesWithProfiles fetches user profiles and merges them with match data
func (s *MatchService) EnrichMatchesWithProfiles(ctx context.Context, userHandle string, matches []models.Match) ([]models.MatchWithProfile, error) {
	var enrichedMatches []models.MatchWithProfile

	for _, match := range matches {
		// Determine the other user handle
		otherUserHandle := match.User1Handle
		if match.User1Handle == userHandle {
			otherUserHandle = match.User2Handle
		}

		// Fetch the other user's profile
		userProfileKey := map[string]types.AttributeValue{
			"userhandle": &types.AttributeValueMemberS{Value: otherUserHandle},
		}

		userProfileItem, err := s.Dynamo.GetItem(ctx, models.UserProfilesTable, userProfileKey)
		if err != nil {
			log.Printf("⚠️ Warning: Failed to fetch profile for %s: %v", otherUserHandle, err)
			continue
		}

		// Convert profile data from DynamoDB to struct
		var userProfileData models.UserProfile
		err = attributevalue.UnmarshalMap(userProfileItem, &userProfileData)
		if err != nil {
			log.Printf("⚠️ Warning: Failed to parse profile data for %s: %v", otherUserHandle, err)
			continue
		}

		// ✅ Merge match and profile data
		combinedData := models.MatchWithProfile{
			MatchID:     match.MatchID,
			User1Handle: match.User1Handle,
			User2Handle: match.User2Handle,
			Status:      match.Status,
			CreatedAt:   match.CreatedAt,

			// Profile Fields of the Other User
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

		enrichedMatches = append(enrichedMatches, combinedData)
	}

	return enrichedMatches, nil
}
