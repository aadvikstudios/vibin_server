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

// FetchMatches retrieves matches for a user (including private and group chats)
func (s *MatchService) FetchMatches(ctx context.Context, userHandle string) ([]models.Match, error) {
	var matches []models.Match

	log.Printf("🔍 Querying matches where userHandle is in Users array: %s", userHandle)

	// Use `Users` array to find matches for the user
	keyCondition := "contains(users, :userHandle)"
	expressionValues := map[string]types.AttributeValue{
		":userHandle": &types.AttributeValueMemberS{Value: userHandle},
	}

	matchItems, err := s.Dynamo.QueryItemsWithOptions(ctx, models.MatchesTable, keyCondition, expressionValues, nil, 100, true)
	if err != nil {
		log.Printf("❌ Error querying Matches table: %v", err)
		return nil, err
	}

	// Unmarshal results
	for _, item := range matchItems {
		var match models.Match
		if err := attributevalue.UnmarshalMap(item, &match); err != nil {
			log.Printf("❌ Error unmarshalling match: %v", err)
			continue
		}
		matches = append(matches, match)
	}

	log.Printf("✅ Found %d matches for userHandle: %s", len(matches), userHandle)
	return matches, nil
}

// Enrich Matches with User Profiles
func (s *MatchService) EnrichMatchesWithProfiles(ctx context.Context, userHandle string, matches []models.MatchWithProfile) ([]models.MatchWithProfile, error) {
	var enrichedMatches []models.MatchWithProfile

	for _, match := range matches {
		// Fetch profiles for all users in the match
		var userProfiles []models.UserProfile
		for _, user := range match.Users {
			if user == userHandle {
				continue // Skip current user
			}

			// Fetch user profile
			userProfileKey := map[string]types.AttributeValue{
				"userhandle": &types.AttributeValueMemberS{Value: user},
			}

			userProfileItem, err := s.Dynamo.GetItem(ctx, models.UserProfilesTable, userProfileKey)
			if err != nil {
				log.Printf("⚠️ Warning: Failed to fetch profile for %s: %v", user, err)
				continue
			}

			// Convert profile data from DynamoDB to struct
			var userProfileData models.UserProfile
			err = attributevalue.UnmarshalMap(userProfileItem, &userProfileData)
			if err != nil {
				log.Printf("⚠️ Warning: Failed to parse profile data for %s: %v", user, err)
				continue
			}

			userProfiles = append(userProfiles, userProfileData)
		}

		// Update match object with profile data
		match.UserProfiles = userProfiles
		enrichedMatches = append(enrichedMatches, match)
	}

	return enrichedMatches, nil
}

// Fetch Matches & Enrich with Profile, Last Message & Unread Status
func (s *MatchService) GetMatchesByUserHandle(ctx context.Context, userHandle string) ([]models.MatchWithProfile, error) {
	// Fetch Matches
	matches, err := s.FetchMatches(ctx, userHandle)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch matches: %w", err)
	}

	// Fetch Last Message & Unread Status for Each Match
	matchesWithMessages, err := s.AttachLastMessageAndUnreadStatus(ctx, userHandle, matches)
	if err != nil {
		return nil, fmt.Errorf("failed to attach last message: %w", err)
	}

	// Fetch User Profiles
	enrichedMatches, err := s.EnrichMatchesWithProfiles(ctx, userHandle, matchesWithMessages)
	if err != nil {
		return nil, fmt.Errorf("failed to enrich matches with profiles: %w", err)
	}

	return enrichedMatches, nil
}

// Attach Last Message & Unread Status for Each Match
func (s *MatchService) AttachLastMessageAndUnreadStatus(ctx context.Context, userHandle string, matches []models.Match) ([]models.MatchWithProfile, error) {
	var enrichedMatches []models.MatchWithProfile

	for _, match := range matches {
		// Query latest message for the match
		lastMessage, isUnread, err := s.FetchLastMessageAndUnread(ctx, match.MatchID, userHandle)
		if err != nil {
			log.Printf("⚠️ Warning: Failed to fetch last message for MatchID %s: %v", match.MatchID, err)
			lastMessage = ""
			isUnread = false
		}

		// Convert match to MatchWithProfile
		enrichedMatch := models.MatchWithProfile{
			MatchID:     match.MatchID,
			Users:       match.Users,
			Type:        match.Type,
			Status:      match.Status,
			CreatedAt:   match.CreatedAt,
			LastMessage: lastMessage,
			IsUnread:    isUnread,
		}

		enrichedMatches = append(enrichedMatches, enrichedMatch)
	}

	return enrichedMatches, nil
}

// Fetch Last Message & Unread Status for a Match
func (s *MatchService) FetchLastMessageAndUnread(ctx context.Context, matchID string, userHandle string) (string, bool, error) {
	log.Printf("🔍 Fetching last message & unread status for MatchID: %s", matchID)

	// Query Latest Message from DynamoDB
	keyCondition := "#matchId = :matchId"
	expressionValues := map[string]types.AttributeValue{
		":matchId": &types.AttributeValueMemberS{Value: matchID},
	}
	expressionNames := map[string]string{
		"#matchId": "matchId",
	}

	messages, err := s.Dynamo.QueryItemsWithOptions(ctx, models.MessagesTable, keyCondition, expressionValues, expressionNames, 1, true)
	if err != nil {
		log.Printf("❌ Error fetching last message for MatchID %s: %v", matchID, err)
		return "", false, err
	}

	if len(messages) == 0 {
		return "", false, nil // No messages found
	}

	// Unmarshal Last Message
	var lastMessage models.Message
	err = attributevalue.UnmarshalMap(messages[0], &lastMessage)
	if err != nil {
		log.Printf("❌ Error unmarshalling last message: %v", err)
		return "", false, err
	}

	// Check Unread Status (if sender is NOT the current user)
	isUnread := lastMessage.IsUnread == "true" && lastMessage.SenderID != userHandle

	log.Printf("✅ Last message: %s, IsUnread: %v", lastMessage.Content, isUnread)
	return lastMessage.Content, isUnread, nil
}
