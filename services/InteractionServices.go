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

// InteractionService handles interactions (like, ping, and matches)
type InteractionService struct {
	Dynamo *DynamoService
}

// SaveInteraction handles "like" and "ping" interactions
func (s *InteractionService) SaveInteraction(ctx context.Context, sender, receiver, interactionType, message string) error {
	sortKey := sender + "#" + interactionType

	interaction := models.Interaction{
		ReceiverHandle: receiver,
		SortKey:        sortKey,
		SenderHandle:   sender,
		Type:           interactionType,
		Status:         "pending",
		CreatedAt:      time.Now().Format(time.RFC3339),
	}

	// Store message only if it's a ping
	if interactionType == "ping" && message != "" {
		interaction.Message = &message
	}

	log.Printf("📥 Saving interaction: %+v", interaction)

	if err := s.Dynamo.PutItem(ctx, models.InteractionsTable, interaction); err != nil {
		log.Printf("❌ Failed to save interaction: %v", err)
		return fmt.Errorf("failed to save interaction: %w", err)
	}

	log.Printf("✅ Interaction recorded: %s -> %s (%s)", sender, receiver, interactionType)

	// Handle match scenario
	if interactionType == "like" {
		isMatch, err := s.IsMatch(ctx, sender, receiver)
		if err != nil {
			log.Printf("⚠️ Error checking for match: %v", err)
			return nil
		}
		if isMatch {
			return s.HandleMatchWithUpdate(ctx, []string{sender, receiver}, "")
		}
	}

	return nil
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

// HandleMatchWithUpdate ensures match creation and updates statuses
func (s *InteractionService) HandleMatchWithUpdate(ctx context.Context, users []string, message string) error {
	log.Printf("🎉 Creating match for users: %v", users)

	// ✅ Update interaction statuses for all users
	for _, user := range users {
		for _, otherUser := range users {
			if user != otherUser {
				if err := s.UpdateInteractionStatus(ctx, user, otherUser, "match", "like"); err != nil {
					log.Printf("⚠️ Error updating status for %s -> %s: %v", user, otherUser, err)
				}
			}
		}
	}

	return s.HandleMatch(ctx, users, message)
}

// ✅ GetLikedOrDislikedUsers now correctly fetches interactions using GSI
func (s *InteractionService) GetLikedOrDislikedUsers(ctx context.Context, senderHandle string) (map[string]bool, error) {
	log.Printf("🔍 Fetching interactions for %s", senderHandle)

	// ✅ Query interactions where senderHandle = senderHandle
	keyCondition := "senderHandle = :sender"
	expressionValues := map[string]types.AttributeValue{
		":sender": &types.AttributeValueMemberS{Value: senderHandle},
	}

	// ✅ Use GSI (senderHandle-index) for efficient querying
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

// UpdateInteractionStatus ensures an existing record is updated instead of inserting a new one
func (s *InteractionService) UpdateInteractionStatus(ctx context.Context, senderHandle, receiverHandle, newStatus, action string) error {
	log.Printf("🔄 Updating interaction status to '%s' for %s -> %s (Action: %s)", newStatus, senderHandle, receiverHandle, action)

	// Determine the correct SortKey based on action type
	var sortKey string
	if action == "like" {
		sortKey = senderHandle + "#like"
	} else if action == "ping" {
		sortKey = senderHandle + "#ping"
	} else {
		log.Printf("⚠️ Unsupported action type: %s", action)
		return fmt.Errorf("unsupported action type: %s", action)
	}

	// Define key for updating the existing interaction record
	key := map[string]types.AttributeValue{
		"receiverHandle": &types.AttributeValueMemberS{Value: receiverHandle},
		"sk":             &types.AttributeValueMemberS{Value: sortKey}, // ✅ Uses dynamically assigned sortKey
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

	log.Printf("✅ Successfully updated interaction status to '%s' for %s -> %s (SortKey: %s)", newStatus, senderHandle, receiverHandle, sortKey)
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

// HandleMatch creates a match and inserts an initial message
func (s *InteractionService) HandleMatch(ctx context.Context, users []string, message string) error {
	log.Printf("🎉 Creating match for users: %v", users)

	matchID, err := s.CreateMatch(ctx, users)
	if err != nil {
		return err
	}

	// Insert initial message if needed
	if message != "" {
		if err := s.SendInitialMessage(ctx, matchID, users[0], message); err != nil {
			return err
		}
	}

	return nil
}

// ✅ Create a Match (Now supports Groups)
func (s *InteractionService) CreateMatch(ctx context.Context, users []string) (string, error) {
	matchID := uuid.New().String()

	match := models.Match{
		MatchID:   matchID,
		Users:     users,     // ✅ Store all participants
		Type:      "private", // Default to private chat
		Status:    "active",
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	if len(users) > 2 {
		match.Type = "group" // ✅ Convert to group chat
	}

	if err := s.Dynamo.PutItem(ctx, models.MatchesTable, match); err != nil {
		log.Printf("❌ Failed to create match: %v", err)
		return "", fmt.Errorf("failed to create match: %w", err)
	}

	return matchID, nil
}

// ✅ Send Initial Message (Now supports multiple users)
func (s *InteractionService) SendInitialMessage(ctx context.Context, matchID, sender, message string) error {
	messageID := uuid.New().String()
	createdAt := time.Now().Format(time.RFC3339)

	newMessage := models.Message{
		MatchID:   matchID,
		MessageID: messageID,
		SenderID:  sender,
		Content:   message,
		IsUnread:  "true",
		Liked:     false,
		CreatedAt: createdAt,
	}

	if err := s.Dynamo.PutItem(ctx, models.MessagesTable, newMessage); err != nil {
		return fmt.Errorf("failed to send initial message: %w", err)
	}

	log.Printf("📩 Initial message sent for Match %s: %s", matchID, message)
	return nil
}

// ✅ Fetch Matches by UserHandle
func (s *InteractionService) FetchMatches(ctx context.Context, userHandle string) ([]models.Match, error) {
	log.Printf("🔍 Fetching matches for %s", userHandle)

	keyCondition := "users CONTAINS :userHandle"
	expressionValues := map[string]types.AttributeValue{
		":userHandle": &types.AttributeValueMemberS{Value: userHandle},
	}

	items, err := s.Dynamo.QueryItems(ctx, models.MatchesTable, keyCondition, expressionValues, nil, 100)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch matches: %w", err)
	}

	var matches []models.Match
	err = attributevalue.UnmarshalListOfMaps(items, &matches)
	if err != nil {
		return nil, err
	}

	log.Printf("✅ Found %d matches for %s", len(matches), userHandle)
	return matches, nil
}

// ✅ Fetch Matches with Profile Enrichment
func (s *InteractionService) GetMatchesWithProfiles(ctx context.Context, userHandle string) ([]models.MatchWithProfile, error) {
	matches, err := s.FetchMatches(ctx, userHandle)
	if err != nil {
		return nil, err
	}

	var enrichedMatches []models.MatchWithProfile
	for _, match := range matches {
		userProfiles := []models.UserProfile{}
		for _, user := range match.Users {
			profile, err := s.FetchUserProfile(ctx, user)
			if err == nil {
				userProfiles = append(userProfiles, profile)
			}
		}
		enrichedMatches = append(enrichedMatches, models.MatchWithProfile{
			MatchID:      match.MatchID,
			Users:        match.Users,
			Type:         match.Type,
			Status:       match.Status,
			CreatedAt:    match.CreatedAt,
			UserProfiles: userProfiles,
		})
	}

	return enrichedMatches, nil
}

// ✅ Fetch a User's Profile from DynamoDB
func (s *InteractionService) FetchUserProfile(ctx context.Context, userHandle string) (models.UserProfile, error) {
	log.Printf("🔍 Fetching profile for user: %s", userHandle)

	key := map[string]types.AttributeValue{
		"userhandle": &types.AttributeValueMemberS{Value: userHandle},
	}

	item, err := s.Dynamo.GetItem(ctx, models.UserProfilesTable, key)
	if err != nil {
		log.Printf("⚠️ Error fetching profile: %v", err)
		return models.UserProfile{}, err
	}

	var profile models.UserProfile
	err = attributevalue.UnmarshalMap(item, &profile)
	if err != nil {
		log.Printf("❌ Error unmarshalling profile: %v", err)
		return models.UserProfile{}, err
	}

	return profile, nil
}
