package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"vibin_server/models"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

// InteractionService handles interactions (like, ping, and matches)
type InteractionService struct {
	Dynamo             *DynamoService
	UserProfileService *UserProfileService
	ChatService        *ChatService
}

// GetInteraction retrieves an interaction between two users
func (s *InteractionService) GetInteraction(ctx context.Context, sender, receiver string) (*models.Interaction, error) {
	log.Printf("🔍 Checking if interaction exists: %s -> %s", sender, receiver)

	key := map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: "USER#" + sender},
		"SK": &types.AttributeValueMemberS{Value: "INTERACTION#" + receiver},
	}

	item, err := s.Dynamo.GetItem(ctx, models.InteractionsTable, key)
	if err != nil {
		if strings.Contains(err.Error(), "item not found") {
			log.Printf("ℹ️ No previous interaction found for %s -> %s. Proceeding to create a new one.", sender, receiver)
			return nil, nil // ✅ This is expected; allow creation of a new interaction
		}
		log.Printf("❌ Unexpected DynamoDB error while fetching interaction: %v", err)
		return nil, err
	}

	if item == nil {
		log.Printf("ℹ️ No interaction record exists for %s -> %s. Creating a new one.", sender, receiver)
		return nil, nil
	}

	var interaction models.Interaction
	err = attributevalue.UnmarshalMap(item, &interaction)
	if err != nil {
		log.Printf("❌ Error unmarshalling interaction: %v", err)
		return nil, err
	}

	return &interaction, nil
}

func (s *InteractionService) CreateOrUpdateInteraction(
	ctx context.Context, sender, receiver, interactionType, action string, message *string) (bool, *models.MatchedUserDetails, error) {

	log.Printf("🔄 Processing %s from %s -> %s", interactionType, sender, receiver)

	// Check if an existing interaction exists
	existingInteraction, err := s.GetInteraction(ctx, sender, receiver)
	if err != nil {
		log.Printf("⚠️ Error fetching interaction: %v", err)
		return false, nil, err
	}

	var newStatus string
	var matchID *string
	isMatch := false // Default value
	var matchedUser *models.MatchedUserDetails

	switch action {
	case "like":
		newStatus = "pending"

		// ✅ Check if it's a mutual match
		isMatch, err = s.CheckMutualMatch(ctx, sender, receiver)
		log.Printf("⚠️ isMatch fetching interaction: %t", isMatch)

		if err != nil {
			return false, nil, err
		}

		// ✅ If mutual match, update status
		if isMatch {
			newStatus = "match"
			matchID, err = s.HandleMutualMatch(ctx, sender, receiver)
			if err != nil {
				return false, nil, err
			}

			// ✅ Fetch receiver's profile
			profile, err := s.UserProfileService.GetUserProfileByHandle(ctx, receiver)
			if err != nil {
				log.Printf("⚠️ Failed to fetch user profile for %s: %v", receiver, err)
			} else {
				log.Printf("✅ Fetched profile for %s: Name=%s, Photos=%v", receiver, profile.Name, profile.Photos)

				photo := ""
				if len(profile.Photos) > 0 {
					photo = profile.Photos[0]
				}

				matchedUser = &models.MatchedUserDetails{
					Name:       profile.Name,
					UserHandle: receiver,
					Photo:      photo,
					MatchID:    *matchID,
				}
				log.Printf("✅ MatchedUserDetails created: %+v", matchedUser)
			}
		}

	case "dislike":
		newStatus = "declined"
	case "ping":
		newStatus = "pending"
	case "approve":
		newStatus = "match"
		isMatch = true
		generatedMatchID := uuid.New().String()
		matchID = &generatedMatchID

		// ✅ Fetch receiver's profile
		profile, err := s.UserProfileService.GetUserProfileByHandle(ctx, receiver)
		if err != nil {
			log.Printf("⚠️ Failed to fetch user profile for %s: %v", receiver, err)
		} else {
			log.Printf("✅ Fetched profile for %s: Name=%s, Photos=%v", receiver, profile.Name, profile.Photos)

			photo := ""
			if len(profile.Photos) > 0 {
				photo = profile.Photos[0]
			}

			matchedUser = &models.MatchedUserDetails{
				Name:       profile.Name,
				UserHandle: receiver,
				Photo:      photo,
				MatchID:    *matchID,
			}
		}
	case "reject":
		newStatus = "rejected"
	default:
		return false, nil, fmt.Errorf("❌ Unsupported interaction type: %s", interactionType)
	}

	// ✅ If the interaction does not exist, create it
	if existingInteraction == nil {
		log.Printf("🆕 No existing interaction found. Creating a new interaction for %s -> %s", sender, receiver)
		err := s.CreateInteraction(ctx, sender, receiver, interactionType, newStatus, matchID, message)
		if err != nil {
			log.Printf("❌ Failed to create interaction: %v", err)
			return false, nil, err
		}
		log.Println("✅ New interaction successfully created.")
		return isMatch, matchedUser, nil
	}

	// ✅ Otherwise, update existing interaction
	err = s.UpdateInteractionStatus(ctx, sender, receiver, newStatus, matchID, message, nil)
	if err != nil {
		return false, nil, err
	}

	return isMatch, matchedUser, nil
}
func (s *InteractionService) HandlePingApproval(ctx context.Context, sender, receiver string) error {
	log.Printf("✅ Handling Ping Approval: %s -> %s", sender, receiver)

	// ✅ Generate a Match ID
	matchID := uuid.New().String()

	// ✅ Fetch existing interaction for sender → receiver
	interactionData, err := s.GetInteraction(ctx, sender, receiver)
	if err != nil {
		log.Printf("❌ Failed to fetch sender interaction: %v", err)
		return err
	}

	// ✅ Ensure interactionType and message exist
	var interactionType, message string
	if interactionData != nil {
		interactionType = interactionData.InteractionType
		if interactionData.Message != nil {
			message = *interactionData.Message
		}
	} else {
		log.Printf("⚠️ No existing interactionType found for %s -> %s", sender, receiver)
		return fmt.Errorf("missing interactionType in sender's record")
	}
	// ✅ Update sender → receiver
	err = s.UpdateInteractionStatus(ctx, sender, receiver, "match", &matchID, &message, nil)
	if err != nil {
		log.Printf("❌ Failed to approve ping: %v", err)
		return err
	}
	// #[TODO] we need create for sender -> reciever instead of create
	// ✅ Update receiver → sender (Now with `interactionType` and `message`)
	err = s.UpdateInteractionStatus(ctx, receiver, sender, "match", &matchID, &message, &interactionType)
	if err != nil {
		log.Printf("⚠️ Failed to update reverse ping status: %v", err)
	}

	// ✅ Send an initial message (with original ping content)
	err = s.CreateInitialMessage(ctx, sender, receiver, matchID, true)
	if err != nil {
		log.Printf("⚠️ Failed to send initial message: %v", err)
	}

	log.Printf("✅ Ping Approved: %s <-> %s", sender, receiver)
	return nil
}

func (s *InteractionService) HandlePingDecline(ctx context.Context, sender, receiver string) error {
	log.Printf("🚫 Handling Ping Decline: %s -> %s", sender, receiver)

	// ✅ Fetch the existing interaction to get `interactionType`
	interactionData, err := s.GetInteraction(ctx, sender, receiver)
	if err != nil {
		log.Printf("❌ Failed to fetch sender interaction: %v", err)
		return err
	}

	// ✅ Ensure interactionType exists
	var interactionType *string
	if interactionData != nil && interactionData.InteractionType != "" {
		interactionType = &interactionData.InteractionType
	} else {
		log.Printf("⚠️ No interactionType found for %s -> %s", sender, receiver)
	}

	// ✅ Update sender → receiver status to "declined"
	err = s.UpdateInteractionStatus(ctx, sender, receiver, "declined", nil, nil, nil)
	if err != nil {
		log.Printf("❌ Failed to decline ping: %v", err)
		return err
	}

	// ✅ Update receiver → sender status to "declined" (Now with `interactionType`)
	err = s.UpdateInteractionStatus(ctx, receiver, sender, "declined", nil, nil, interactionType)
	if err != nil {
		log.Printf("⚠️ Failed to update reverse ping status: %v", err)
	}

	log.Printf("✅ Ping Declined: %s -> %s", sender, receiver)
	return nil
}

func (s *InteractionService) CheckMutualMatch(ctx context.Context, sender, receiver string) (bool, error) {
	log.Printf("🔍 Checking for mutual match: %s <-> %s", sender, receiver)

	// Fetch existing interaction (if any) where receiver liked sender
	mutualLike, err := s.GetInteraction(ctx, receiver, sender)
	if err != nil {
		log.Printf("❌ Error fetching interaction for mutual match check: %v", err)
		return false, err
	}

	// ✅ If the receiver also liked the sender, it's a mutual match
	if mutualLike != nil && mutualLike.Status == "pending" {
		log.Printf("🔥 Mutual Match Found! %s <-> %s", sender, receiver)
		return true, nil
	}

	// ❌ No mutual match
	return false, nil
}
func (s *InteractionService) HandleMutualMatch(ctx context.Context, sender, receiver string) (*string, error) {
	log.Printf("🔄 Handling mutual match update for: %s <-> %s", sender, receiver)

	// Generate a match ID
	matchID := uuid.New().String()

	// ✅ Update UserB -> UserA interaction to "match"
	err := s.UpdateInteractionStatus(ctx, receiver, sender, "match", &matchID, nil, nil)
	if err != nil {
		log.Printf("❌ Failed to update mutual match for %s -> %s: %v", receiver, sender, err)
		return nil, err
	}

	// ✅ Create an initial message (default congratulatory message)
	err = s.CreateInitialMessage(ctx, sender, receiver, matchID, false)
	if err != nil {
		log.Printf("⚠️ Failed to send initial message for match %s: %v", matchID, err)
	}

	return &matchID, nil
}

func (s *InteractionService) CreateInitialMessage(ctx context.Context, sender, receiver, matchID string, isPing bool) error {
	log.Printf("💬 Creating initial message for matchId: %s between %s & %s", matchID, sender, receiver)

	// Determine message content and sender
	var content string
	var originalSender string

	if isPing {
		// ✅ Fetch the original ping interaction to get the message content
		originalInteraction, err := s.GetInteraction(ctx, sender, receiver)
		if err != nil {
			log.Printf("❌ Failed to fetch original ping interaction: %v", err)
			return err
		}

		if originalInteraction == nil || originalInteraction.Message == nil {
			log.Printf("⚠️ No original ping message found, using default content")
			content = "Hey! I sent you a ping. Let's connect! 😊"
		} else {
			content = *originalInteraction.Message // ✅ Use original ping message
		}

		originalSender = sender // ✅ Keep the original sender
	} else {
		// ✅ Default message for mutual like
		content = "Congratulations! You both liked each other. Say hello! 👋"
		originalSender = sender
	}

	// ✅ Define the first message
	initialMessage := models.Message{
		MatchID:   matchID,
		MessageID: uuid.New().String(),
		SenderID:  originalSender, // ✅ Keep the original sender
		Content:   content,
		CreatedAt: time.Now().Format(time.RFC3339), // Store timestamp
		Liked:     false,
	}

	// ✅ Set IsUnread using helper method
	initialMessage.SetIsUnread(true)

	// ✅ Send message using ChatService
	err := s.ChatService.SendMessage(ctx, initialMessage)
	if err != nil {
		log.Printf("❌ Failed to send initial message: %v", err)
		return err
	}

	log.Printf("✅ Initial message sent successfully for matchId: %s", matchID)
	return nil
}

// CreateInteraction inserts a new interaction into DynamoDB
func (s *InteractionService) CreateInteraction(ctx context.Context, sender, receiver, interactionType, status string, matchID *string, message *string) error {
	log.Printf("🆕 Creating a new interaction for %s -> %s", sender, receiver)

	now := time.Now().Format(time.RFC3339)
	interaction := models.Interaction{
		PK:              "USER#" + sender,
		SK:              "INTERACTION#" + receiver,
		SenderHandle:    sender,
		ReceiverHandle:  receiver,
		InteractionType: interactionType,
		Status:          status,
		MatchID:         matchID,
		Message:         message,
		CreatedAt:       now,
		LastUpdated:     now,
	}

	log.Printf("📥 Saving new interaction: %+v", interaction)
	err := s.Dynamo.PutItem(ctx, models.InteractionsTable, interaction)
	if err != nil {
		log.Printf("❌ Error inserting interaction: %v", err)
		return fmt.Errorf("failed to create interaction: %w", err)
	}
	log.Println("✅ Interaction successfully created.")
	return nil
}

// UpdateInteractionStatus updates the status of an existing interaction and ensures all fields are properly set
func (s *InteractionService) UpdateInteractionStatus(ctx context.Context, sender, receiver, newStatus string, matchID, message, interactionType *string) error {
	log.Printf("🔄 Updating interaction %s -> %s to status: %s", sender, receiver, newStatus)

	updateExpression := "SET #status = :status, #lastUpdated = :lastUpdated, #senderHandle = :sender, #receiverHandle = :receiver"
	expressionValues := map[string]types.AttributeValue{
		":status":      &types.AttributeValueMemberS{Value: newStatus},
		":lastUpdated": &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
		":sender":      &types.AttributeValueMemberS{Value: sender},   // ✅ Directly using sender
		":receiver":    &types.AttributeValueMemberS{Value: receiver}, // ✅ Directly using receiver
	}
	expressionNames := map[string]string{
		"#status":         "status",
		"#lastUpdated":    "lastUpdated",
		"#senderHandle":   "senderHandle",
		"#receiverHandle": "receiverHandle",
	}

	// Add MatchID if provided
	if matchID != nil {
		updateExpression += ", #matchId = :matchId"
		expressionValues[":matchId"] = &types.AttributeValueMemberS{Value: *matchID}
		expressionNames["#matchId"] = "matchId"
	}

	// Add message if provided
	if message != nil {
		updateExpression += ", #message = :message"
		expressionValues[":message"] = &types.AttributeValueMemberS{Value: *message}
		expressionNames["#message"] = "message"
	}

	// Add interactionType if provided
	if interactionType != nil {
		updateExpression += ", #interactionType = :interactionType"
		expressionValues[":interactionType"] = &types.AttributeValueMemberS{Value: *interactionType}
		expressionNames["#interactionType"] = "interactionType"
	}

	// Define key for update
	key := map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: "USER#" + sender},
		"SK": &types.AttributeValueMemberS{Value: "INTERACTION#" + receiver},
	}

	// Execute update
	_, err := s.Dynamo.UpdateItem(ctx, models.InteractionsTable, updateExpression, key, expressionValues, expressionNames)
	if err != nil {
		log.Printf("❌ Error updating interaction status: %v", err)
		return err
	}

	log.Println("✅ Interaction status successfully updated.")
	return nil
}

func (s *InteractionService) GetMutualMatches(ctx context.Context, userHandle string) ([]models.MatchedUserDetailsForConnections, error) {
	log.Printf("🔍 Fetching mutual matches for user: %s", userHandle)

	// Define the Global Secondary Index (GSI) for querying matches
	indexName := "status-index" // Ensure this is correctly configured in DynamoDB
	keyCondition := "#PK = :user AND #status = :matchStatus"

	expressionValues := map[string]types.AttributeValue{
		":user":        &types.AttributeValueMemberS{Value: "USER#" + userHandle},
		":matchStatus": &types.AttributeValueMemberS{Value: "match"},
	}

	expressionNames := map[string]string{
		"#PK":     "PK",
		"#status": "status",
	}

	// 🔍 Query DynamoDB for mutual matches
	items, err := s.Dynamo.QueryItemsWithIndex(ctx, models.InteractionsTable, indexName, keyCondition, expressionValues, expressionNames, 100)
	if err != nil {
		log.Printf("❌ Error fetching mutual matches from DynamoDB: %v", err)
		return nil, fmt.Errorf("failed to fetch matches: %w", err)
	}

	if len(items) == 0 {
		log.Printf("⚠️ No mutual matches found for user: %s", userHandle)
		return []models.MatchedUserDetailsForConnections{}, nil
	}

	var matchesWithDetails []models.MatchedUserDetailsForConnections

	// Process each interaction record
	for _, item := range items {
		var interaction models.Interaction
		err := attributevalue.UnmarshalMap(item, &interaction)
		if err != nil {
			log.Printf("⚠️ Skipping item due to unmarshalling error: %v", err)
			continue
		}

		// Determine which handle to fetch profile for
		matchedUserHandle := interaction.ReceiverHandle
		if matchedUserHandle == userHandle {
			matchedUserHandle = interaction.SenderHandle // Reverse if needed
		}

		// 🔍 Fetch user profile for the matched user
		profile, err := s.UserProfileService.GetUserProfileByHandle(ctx, matchedUserHandle)
		if err != nil {
			log.Printf("⚠️ Failed to fetch profile for %s: %v", matchedUserHandle, err)
			continue
		}

		photo := ""
		if profile.Photos != nil && len(profile.Photos) > 0 {
			photo = profile.Photos[0]
		}

		// 🔍 Fetch last message for the match
		lastMessage, err := s.ChatService.GetLastMessageByMatchID(ctx, *interaction.MatchID)
		if err != nil {
			log.Printf("⚠️ Error fetching last message for matchId: %s: %v", *interaction.MatchID, err)
		}

		// Default values for last message fields
		lastMessageText := ""
		lastMessageSender := ""
		lastMessageIsRead := true

		if lastMessage != nil {
			lastMessageText = lastMessage.Content
			lastMessageSender = lastMessage.SenderID
			lastMessageIsRead = lastMessage.IsUnread == "false"
		}

		// ✅ Append to results with all details
		matchesWithDetails = append(matchesWithDetails, models.MatchedUserDetailsForConnections{
			Name:              profile.Name,
			UserHandle:        profile.UserHandle,
			MatchID:           *interaction.MatchID,
			Photo:             photo,
			LastMessage:       lastMessageText,
			LastMessageSender: lastMessageSender,
			LastMessageIsRead: lastMessageIsRead,
		})
	}

	log.Printf("✅ Found %d mutual matches with last messages for %s", len(matchesWithDetails), userHandle)
	return matchesWithDetails, nil
}

func (s *InteractionService) GetInteractedUsers(ctx context.Context, userHandle string, interactionTypes []string) ([]string, error) {
	log.Printf("🔍 Fetching interacted users for: %s with types: %v", userHandle, interactionTypes)

	// ✅ Ensure the correct GSI name is used
	indexName := models.InteractionTypeIndex

	// ✅ Use 'interactionType' in KeyConditionExpression (Not FilterExpression)
	var keyConditions []string
	expressionValues := map[string]types.AttributeValue{
		":userHandle": &types.AttributeValueMemberS{Value: "USER#" + userHandle},
	}
	expressionNames := map[string]string{
		"#PK":              "PK",
		"#interactionType": "interactionType",
	}

	// ✅ KeyConditionExpression supports "IN" only if it's a Sort Key
	if len(interactionTypes) == 1 {
		expressionValues[":interactionType"] = &types.AttributeValueMemberS{Value: interactionTypes[0]}
		keyConditions = append(keyConditions, "#PK = :userHandle AND #interactionType = :interactionType")
	} else {
		// ✅ Use "OR" alternative: Query multiple times if needed
		var interactedUsers []string
		for _, interactionType := range interactionTypes {
			log.Printf("🔄 Querying for interaction type: %s", interactionType)

			tempExpressionValues := map[string]types.AttributeValue{
				":userHandle":      expressionValues[":userHandle"],
				":interactionType": &types.AttributeValueMemberS{Value: interactionType},
			}

			items, err := s.Dynamo.QueryItemsWithIndex(
				ctx, models.InteractionsTable, indexName,
				"#PK = :userHandle AND #interactionType = :interactionType",
				tempExpressionValues, expressionNames, 50,
			)
			if err != nil {
				log.Printf("❌ Error querying interactionType '%s': %v", interactionType, err)
				continue // Skip this type but continue others
			}

			for _, item := range items {
				var interaction models.Interaction
				if err := attributevalue.UnmarshalMap(item, &interaction); err == nil {
					interactedUsers = append(interactedUsers, interaction.ReceiverHandle)
				}
			}
		}
		log.Printf("✅ Total Interacted Users Found: %d", len(interactedUsers))
		return interactedUsers, nil
	}

	// ✅ Query with the correct key conditions
	log.Printf("🔍 Querying GSI '%s' with condition: %s", indexName, keyConditions[0])
	items, err := s.Dynamo.QueryItemsWithIndex(ctx, models.InteractionsTable, indexName, keyConditions[0], expressionValues, expressionNames, 50)
	if err != nil {
		log.Printf("❌ Error querying interacted users: %v", err)
		return nil, fmt.Errorf("failed to fetch interacted users: %w", err)
	}

	// ✅ Extract interacted user handles
	users := []string{}
	for _, item := range items {
		var interaction models.Interaction
		if err := attributevalue.UnmarshalMap(item, &interaction); err == nil {
			users = append(users, interaction.ReceiverHandle)
		}
	}

	log.Printf("✅ Found %d interacted users for %s", len(users), userHandle)
	return users, nil
}

func (s *InteractionService) GetUserInteractions(ctx context.Context, userHandle string) ([]models.InteractionWithProfile, error) {
	log.Printf("🔍 Fetching interactions SENT by user: %s", userHandle)

	keyCondition := "PK = :user"
	expressionValues := map[string]types.AttributeValue{
		":user": &types.AttributeValueMemberS{Value: "USER#" + userHandle},
	}

	items, err := s.Dynamo.QueryItems(ctx, models.InteractionsTable, keyCondition, expressionValues, nil, 100)
	if err != nil {
		log.Printf("❌ Error querying interactions: %v", err)
		return nil, fmt.Errorf("failed to fetch interactions: %w", err)
	}

	var interactionsWithProfiles []models.InteractionWithProfile

	for _, item := range items {
		var interaction models.Interaction
		err := attributevalue.UnmarshalMap(item, &interaction)
		if err != nil {
			log.Printf("⚠️ Skipping item due to unmarshalling error: %v", err)
			continue
		}

		// Fetch user profile for receiver
		profile, err := s.UserProfileService.GetUserProfileByHandle(ctx, interaction.ReceiverHandle)
		if err != nil {
			log.Printf("⚠️ Failed to fetch profile for %s: %v", interaction.ReceiverHandle, err)
			continue
		}

		// Append only selected fields
		interactionsWithProfiles = append(interactionsWithProfiles, models.InteractionWithProfile{
			ReceiverHandle:  interaction.ReceiverHandle,
			SenderHandle:    interaction.SenderHandle,
			InteractionType: interaction.InteractionType,
			Message:         *interaction.Message,
			Status:          interaction.Status,
			CreatedAt:       interaction.CreatedAt,

			// Extracted profile fields
			Name:        profile.Name,
			Age:         profile.Age,
			Gender:      profile.Gender,
			Orientation: profile.Orientation,
			LookingFor:  profile.LookingFor,
			Photos:      profile.Photos,
			Bio:         profile.Bio,
			Interests:   profile.Interests,
		})
	}

	log.Printf("✅ Found %d interactions sent by %s", len(interactionsWithProfiles), userHandle)
	return interactionsWithProfiles, nil
}

func (s *InteractionService) GetReceivedInteractions(ctx context.Context, userHandle string) ([]models.InteractionWithProfile, error) {
	log.Printf("🔍 Fetching interactions RECEIVED by user: %s", userHandle)

	indexName := models.ReceiverHandleIndex
	keyCondition := "#receiverHandle = :receiver"

	expressionValues := map[string]types.AttributeValue{
		":receiver": &types.AttributeValueMemberS{Value: userHandle},
	}
	expressionNames := map[string]string{"#receiverHandle": "receiverHandle"}

	items, err := s.Dynamo.QueryItemsWithIndex(ctx, models.InteractionsTable, indexName, keyCondition, expressionValues, expressionNames, 100)
	if err != nil {
		log.Printf("❌ Error querying received interactions: %v", err)
		return nil, fmt.Errorf("failed to fetch received interactions: %w", err)
	}

	var interactionsWithProfiles []models.InteractionWithProfile

	for _, item := range items {
		var interaction models.Interaction
		err := attributevalue.UnmarshalMap(item, &interaction)
		if err != nil {
			log.Printf("⚠️ Skipping item due to unmarshalling error: %v", err)
			continue
		}

		// Fetch sender's profile
		profile, err := s.UserProfileService.GetUserProfileByHandle(ctx, interaction.SenderHandle)
		if err != nil {
			log.Printf("⚠️ Failed to fetch profile for %s: %v", interaction.SenderHandle, err)
			continue
		}

		interactionsWithProfiles = append(interactionsWithProfiles, models.InteractionWithProfile{
			ReceiverHandle:  interaction.ReceiverHandle,
			SenderHandle:    interaction.SenderHandle,
			InteractionType: interaction.InteractionType,
			Message: func() string {
				if interaction.Message != nil {
					return *interaction.Message
				}
				return ""
			}(),
			Status:    interaction.Status,
			CreatedAt: interaction.CreatedAt,

			// Extracted profile fields
			Name:        profile.Name,
			Age:         profile.Age,
			Gender:      profile.Gender,
			Orientation: profile.Orientation,
			LookingFor:  profile.LookingFor,
			Photos:      profile.Photos,
			Bio:         profile.Bio,
			Interests:   profile.Interests,
		})
	}

	log.Printf("✅ Found %d received interactions for %s", len(interactionsWithProfiles), userHandle)
	return interactionsWithProfiles, nil
}
