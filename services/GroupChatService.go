package services

import (
	"context"
	"fmt"
	"log"
	"vibin_server/models"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// GroupChatService struct
type GroupChatService struct {
	Dynamo *DynamoService
}

// CreateGroupMessage stores a new group message in the GroupMessages table
func (s *GroupChatService) CreateGroupMessage(ctx context.Context, message models.GroupMessage) error {
	log.Printf("📩 Storing group message: %+v", message)

	// ✅ Save message to DynamoDB
	err := s.Dynamo.PutItem(ctx, models.GroupMessageTable, message)
	if err != nil {
		log.Printf("❌ Failed to store group message: %v", err)
		return fmt.Errorf("failed to store group message: %w", err)
	}

	log.Printf("✅ Group message stored successfully")
	return nil
}

// GetMessagesByGroupID fetches the latest messages for a given groupId sorted by createdAt (latest first),
// then reverses the order before returning, so the latest message appears at the bottom in UI.
func (s *GroupChatService) GetMessagesByGroupID(ctx context.Context, groupID string, limit int) ([]models.GroupMessage, error) {
	log.Printf("🔍 Fetching latest %d messages for groupId: %s", limit, groupID)

	// ✅ Define key condition expression for filtering by groupId
	keyCondition := "groupId = :groupId"
	expressionValues := map[string]types.AttributeValue{
		":groupId": &types.AttributeValueMemberS{Value: groupID},
	}

	// ✅ Query DynamoDB (Retrieve latest messages first)
	items, err := s.Dynamo.QueryItemsWithOptions(ctx, models.GroupMessageTable, keyCondition, expressionValues, nil, int32(limit), true)
	if err != nil {
		log.Printf("❌ Error querying group messages: %v", err)
		return nil, fmt.Errorf("failed to fetch group messages: %w", err)
	}

	// ✅ Unmarshal results
	var messages []models.GroupMessage
	err = attributevalue.UnmarshalListOfMaps(items, &messages)
	if err != nil {
		log.Printf("❌ Error unmarshalling group messages: %v", err)
		return nil, fmt.Errorf("failed to parse group messages: %w", err)
	}

	// ✅ Reverse the messages so latest appears at the bottom in UI
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	log.Printf("✅ Found %d messages for groupId: %s, returning in UI-friendly order", len(messages), groupID)
	return messages, nil
}

// MarkGroupMessageAsRead updates the read status of a message for a specific user
func (s *GroupChatService) MarkGroupMessageAsRead(ctx context.Context, groupID, createdAt, userID string) error {
	log.Printf("🔄 Marking message as read for groupId: %s, createdAt: %s by user: %s", groupID, createdAt, userID)

	// ✅ Define update key
	key := map[string]types.AttributeValue{
		"groupId":   &types.AttributeValueMemberS{Value: groupID},
		"createdAt": &types.AttributeValueMemberS{Value: createdAt},
	}

	// ✅ Update `isRead` map for the user
	updateExpression := "SET isRead.#userId = :true, readCount = readCount + :increment"
	expressionValues := map[string]types.AttributeValue{
		":true":      &types.AttributeValueMemberBOOL{Value: true},
		":increment": &types.AttributeValueMemberN{Value: "1"},
	}
	expressionNames := map[string]string{
		"#userId": userID,
	}

	// ✅ Perform update
	_, err := s.Dynamo.UpdateItem(ctx, models.GroupMessageTable, updateExpression, key, expressionValues, expressionNames)
	if err != nil {
		log.Printf("❌ Failed to update read status: %v", err)
		return fmt.Errorf("failed to update read status: %w", err)
	}

	log.Printf("✅ Message marked as read by %s", userID)
	return nil
}

// LikeGroupMessage allows a user to like or unlike a message
func (s *GroupChatService) LikeGroupMessage(ctx context.Context, groupID, createdAt, userID string) error {
	log.Printf("💖 Updating like status for message at %s in groupId: %s by user: %s", createdAt, groupID, userID)

	// ✅ Define update key
	key := map[string]types.AttributeValue{
		"groupId":   &types.AttributeValueMemberS{Value: groupID},
		"createdAt": &types.AttributeValueMemberS{Value: createdAt},
	}

	// ✅ Fetch current message to check if user already liked it
	item, err := s.Dynamo.GetItem(ctx, models.GroupMessageTable, key)
	if err != nil {
		log.Printf("❌ Error fetching message: %v", err)
		return fmt.Errorf("failed to fetch message: %w", err)
	}

	var message models.GroupMessage
	err = attributevalue.UnmarshalMap(item, &message)
	if err != nil {
		log.Printf("❌ Error unmarshalling message: %v", err)
		return fmt.Errorf("failed to parse message: %w", err)
	}

	// ✅ Toggle like status
	liked := false
	if _, exists := message.Likes[userID]; !exists {
		liked = true
	}

	// ✅ Update `likes` map
	updateExpression := "SET likes.#userId = :liked, likeCount = likeCount + :increment"
	expressionValues := map[string]types.AttributeValue{
		":liked": &types.AttributeValueMemberBOOL{Value: liked},
		":increment": &types.AttributeValueMemberN{
			Value: "1",
		},
	}
	if !liked {
		updateExpression = "REMOVE likes.#userId SET likeCount = likeCount - :decrement"
		expressionValues[":decrement"] = &types.AttributeValueMemberN{Value: "1"}
	}

	expressionNames := map[string]string{
		"#userId": userID,
	}

	// ✅ Perform update
	_, err = s.Dynamo.UpdateItem(ctx, models.GroupMessageTable, updateExpression, key, expressionValues, expressionNames)
	if err != nil {
		log.Printf("❌ Failed to update like status: %v", err)
		return fmt.Errorf("failed to update like status: %w", err)
	}

	log.Printf("✅ Successfully updated like status for message at %s", createdAt)
	return nil
}

// GetLastMessageByGroupID fetches the most recent message in a group
func (s *GroupChatService) GetLastMessageByGroupID(ctx context.Context, groupID string) (*models.GroupMessage, error) {
	log.Printf("🔍 Fetching last message for groupId: %s", groupID)

	// ✅ Define key condition
	keyCondition := "groupId = :groupId"
	expressionValues := map[string]types.AttributeValue{
		":groupId": &types.AttributeValueMemberS{Value: groupID},
	}

	// ✅ Query DynamoDB for the latest message
	items, err := s.Dynamo.QueryItemsWithOptions(ctx, models.GroupMessageTable, keyCondition, expressionValues, nil, 1, true) // Descending order
	if err != nil {
		log.Printf("❌ Error fetching last message: %v", err)
		return nil, fmt.Errorf("failed to fetch last message: %w", err)
	}

	// ✅ If no message is found, return nil
	if len(items) == 0 {
		log.Printf("ℹ️ No messages found for groupId: %s", groupID)
		return nil, nil
	}

	// ✅ Unmarshal the most recent message
	var lastMessage models.GroupMessage
	err = attributevalue.UnmarshalMap(items[0], &lastMessage)
	if err != nil {
		log.Printf("❌ Error unmarshalling last message: %v", err)
		return nil, fmt.Errorf("failed to parse last message: %w", err)
	}

	log.Printf("✅ Last message for groupId %s: %+v", groupID, lastMessage)
	return &lastMessage, nil
}
