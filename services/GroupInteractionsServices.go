package services

import (
	"context"
	"errors"
	"log"
	"time"
	"vibin_server/models"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

// GroupInteractionService handles operations related to group invites and approvals
type GroupInteractionService struct {
	Dynamo             *DynamoService
	UserProfileService *UserProfileService
}

// ✅ CreateGroupInvite - Adds a new group invite to DynamoDB after validating the InviteeHandle
func (s *GroupInteractionService) CreateGroupInvite(ctx context.Context, invite models.GroupInteraction) error {
	log.Printf("🔍 Validating invitee handle: %s", invite.InviteeHandle)

	// ✅ Step 1: Validate InviteeHandle (Check if user exists)
	isAvailable, err := s.UserProfileService.IsUserHandleAvailable(ctx, invite.InviteeHandle)
	if err != nil {
		log.Printf("❌ Failed to validate invitee handle '%s': %v", invite.InviteeHandle, err)
		return errors.New("failed to validate invitee handle") // Keep it generic for logging purposes
	}

	// If the handle is available (i.e., user does not exist), reject the invite
	if isAvailable {
		log.Printf("🚫 Invalid invitee handle: '%s' does not exist in the system", invite.InviteeHandle)
		return errors.New("invalid_invitee_handle") // Use a specific error for better handling in the controller
	}

	// ✅ Step 2: Store the invite in DynamoDB (only if validation succeeds)
	log.Printf("✅ Invitee handle '%s' is valid. Proceeding to store the invite in DynamoDB.", invite.InviteeHandle)
	err = s.Dynamo.PutItem(ctx, models.GroupInteractionsTable, invite)
	if err != nil {
		log.Printf("❌ Failed to store group invite for '%s' in DynamoDB: %v", invite.InviteeHandle, err)
		return errors.New("failed to store group invite")
	}

	log.Printf("✅ Successfully stored group invite for '%s' in DynamoDB.", invite.InviteeHandle)
	return nil
}

// ✅ GetSentInvites - Fetches invites created by User A
func (s *GroupInteractionService) GetSentInvites(ctx context.Context, userHandle string) ([]models.GroupInteraction, error) {
	return s.queryGroupInteractions(ctx, "USER#"+userHandle)
}

func (s *GroupInteractionService) GetPendingApprovals(ctx context.Context, approverHandle string) ([]models.GroupInteraction, error) {
	log.Printf("🔍 Fetching pending approvals for approverHandle: %s", approverHandle)

	keyCondition := "approverHandle = :approver AND #status = :status"
	expressionValues := map[string]types.AttributeValue{
		":approver": &types.AttributeValueMemberS{Value: approverHandle},
		":status":   &types.AttributeValueMemberS{Value: "pending"},
	}

	// ✅ Define Expression Attribute Names to handle reserved keywords
	expressionNames := map[string]string{
		"#status": "status",
	}

	log.Printf("📌 DynamoDB Query - Table: %s, Index: %s, KeyCondition: %s, Values: %+v",
		models.GroupInteractionsTable, models.ApprovalIndex, keyCondition, expressionValues)

	// ✅ Query DynamoDB
	items, err := s.Dynamo.QueryItemsWithIndex(ctx, models.GroupInteractionsTable, models.ApprovalIndex, keyCondition, expressionValues, expressionNames, 100)
	if err != nil {
		log.Printf("❌ Error querying DynamoDB: %v", err)
		return nil, err
	}

	log.Printf("✅ Query successful. Items retrieved: %d", len(items))

	var pendingInvites []models.GroupInteraction
	if err := attributevalue.UnmarshalListOfMaps(items, &pendingInvites); err != nil {
		log.Printf("❌ Error unmarshaling DynamoDB items: %v", err)
		return nil, err
	}

	// ✅ Fetch user profiles for invitees
	for i, invite := range pendingInvites {
		inviteeHandle := invite.InviteeHandle

		// Fetch profile for each invitee
		profile, err := s.UserProfileService.GetUserProfileByHandle(ctx, inviteeHandle)
		if err != nil {
			log.Printf("⚠️ Failed to fetch user profile for %s: %v", inviteeHandle, err)
			continue // Skip this invitee if profile fetch fails
		}

		// Extract photo
		photo := ""
		if len(profile.Photos) > 0 {
			photo = profile.Photos[0]
		}

		// Populate InviteeUserDetails
		invite.InviteeProfile = &models.InviteeUserDetails{
			Name:        profile.Name,
			Photo:       photo,
			Bio:         profile.Bio,
			Desires:     profile.Desires,
			Gender:      profile.Gender,
			Interests:   profile.Interests,
			LookingFor:  profile.LookingFor,
			Orientation: profile.Orientation,
		}

		log.Printf("✅ Fetched user profile for invitee %s: %+v", inviteeHandle, invite.InviteeProfile)

		// Update the invite entry
		pendingInvites[i] = invite
	}

	log.Printf("✅ Successfully retrieved %d pending invites with enriched invitee profiles", len(pendingInvites))
	return pendingInvites, nil
}

// ✅ ApproveOrDeclineInvite - Approves or declines a pending invite
func (s *GroupInteractionService) ApproveOrDeclineInvite(ctx context.Context, approverHandle, inviterHandle, inviteeHandle, status string) error {
	log.Printf("🔍 ApproveOrDeclineInvite: Processing request for Approver: %s, Inviter: %s, Invitee: %s, Status: %s", approverHandle, inviterHandle, inviteeHandle, status)

	// ✅ Validate status
	if status != "approved" && status != "declined" {
		log.Printf("❌ Invalid status value: %s. Expected 'approved' or 'declined'.", status)
		return errors.New("invalid status value")
	}

	// ✅ Fetch the existing invite
	pk := "USER#" + inviterHandle
	sk := "GROUP_INVITE#" + inviteeHandle

	log.Printf("📌 Fetching pending invite from GroupInteractions - PK: %s, SK: %s", pk, sk)
	invite, err := s.getGroupInteraction(ctx, pk, sk)
	if err != nil {
		log.Printf("❌ Error fetching invite for Inviter: %s, Invitee: %s - Error: %v", inviterHandle, inviteeHandle, err)
		return err
	}
	if invite == nil {
		log.Printf("⚠️ Invite not found for Inviter: %s, Invitee: %s", inviterHandle, inviteeHandle)
		return errors.New("invite not found")
	}

	log.Printf("✅ Invite found: %+v", invite)

	// ✅ If approved, generate a group ID (if not already present)
	var groupId *string
	if invite.GroupID != nil {
		groupId = invite.GroupID
	} else if status == "approved" {
		newGroupId := uuid.New().String()
		groupId = &newGroupId
		log.Printf("✅ Approved! Assigning new GroupID: %s", *groupId)
	}

	// ✅ Update the invite status
	invite.Status = status
	invite.GroupID = groupId
	invite.LastUpdated = time.Now()

	log.Printf("📤 Saving updated invite in DynamoDB: %+v", invite)
	if err := s.updateGroupInteraction(ctx, *invite); err != nil {
		log.Printf("❌ Error updating invite in DynamoDB: %v", err)
		return err
	}

	// ✅ If declined, return early
	if status == "declined" {
		log.Printf("🚫 Invite declined. No group record created.")
		return nil
	}

	// ✅ Create separate records for Approver, Inviter, and Invitee
	members := []string{approverHandle, inviterHandle, inviteeHandle}

	// ✅ Prepare batch write request
	var groupRecords []models.GroupInteraction
	for _, member := range members {
		groupRecords = append(groupRecords, models.GroupInteraction{
			PK:              "USER#" + member,
			SK:              "GROUP#" + *groupId,
			InteractionType: "group_chat",
			Status:          "active",
			GroupID:         groupId,
			InviterHandle:   inviterHandle,
			ApproverHandle:  approverHandle,
			InviteeHandle:   inviteeHandle,
			Members:         members,
			CreatedAt:       time.Now(),
			LastUpdated:     time.Now(),
		})
	}

	log.Printf("📌 Creating group records for Approver, Inviter, and Invitee")
	if err := s.createBatchGroupInteractions(ctx, groupRecords); err != nil {
		log.Printf("❌ Error creating group records: %v", err)
		return err
	}

	log.Printf("✅ Successfully processed invite for Approver: %s, Inviter: %s, Invitee: %s with Status: %s", approverHandle, inviterHandle, inviteeHandle, status)
	return nil
}

func (s *GroupInteractionService) GetActiveGroups(ctx context.Context, userHandle string) ([]models.GroupInteraction, error) {
	log.Printf("🔍 Searching for active groups where user '%s' is a participant", userHandle)

	// ✅ Query groups for the given user handle
	keyCondition := "PK = :pk"
	expressionValues := map[string]types.AttributeValue{
		":pk": &types.AttributeValueMemberS{Value: "USER#" + userHandle},
	}

	// 🔍 Query DynamoDB
	items, err := s.Dynamo.QueryItems(ctx, models.GroupInteractionsTable, keyCondition, expressionValues, nil, 100)
	if err != nil {
		log.Printf("❌ Error querying active groups for user '%s': %v", userHandle, err)
		return nil, err
	}

	// ✅ Convert to Go struct
	var allGroups []models.GroupInteraction
	if err := attributevalue.UnmarshalListOfMaps(items, &allGroups); err != nil {
		log.Printf("❌ Error unmarshaling groups for '%s': %v", userHandle, err)
		return nil, err
	}

	// ✅ Step 3: Filter for active `group_chat` interactions
	var activeGroups []models.GroupInteraction
	for _, group := range allGroups {
		log.Printf("🔍 Checking group %s with status %s", group.GroupID, group.Status)

		// ✅ Directly use `group.Members` (it is already []string)
		if group.Status == "active" &&
			group.InteractionType == "group_chat" &&
			contains(group.Members, userHandle) { // Directly check in []string
			activeGroups = append(activeGroups, group)
		}
	}

	log.Printf("✅ Found %d active groups for user '%s'", len(activeGroups), userHandle)
	return activeGroups, nil
}

///// 🔹🔹🔹 Helper Methods 🔹🔹🔹 /////

// ✅ queryGroupInteractions - Fetches group interactions for a given user
func (s *GroupInteractionService) queryGroupInteractions(ctx context.Context, partitionKey string) ([]models.GroupInteraction, error) {
	keyCondition := "PK = :pk"
	expressionValues := map[string]types.AttributeValue{
		":pk": &types.AttributeValueMemberS{Value: partitionKey},
	}

	items, err := s.Dynamo.QueryItems(ctx, models.GroupInteractionsTable, keyCondition, expressionValues, nil, 0)
	if err != nil {
		return nil, err
	}

	var interactions []models.GroupInteraction
	if err := attributevalue.UnmarshalListOfMaps(items, &interactions); err != nil {
		return nil, err
	}

	return interactions, nil
}

// ✅ getGroupInteraction - Fetches a single group interaction from DynamoDB
func (s *GroupInteractionService) getGroupInteraction(ctx context.Context, pk, sk string) (*models.GroupInteraction, error) {
	key := map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: pk},
		"SK": &types.AttributeValueMemberS{Value: sk},
	}

	item, err := s.Dynamo.GetItem(ctx, models.GroupInteractionsTable, key)
	if err != nil {
		return nil, err
	}

	if item == nil {
		return nil, errors.New("group interaction not found")
	}

	var interaction models.GroupInteraction
	if err := attributevalue.UnmarshalMap(item, &interaction); err != nil {
		return nil, err
	}

	return &interaction, nil
}

// ✅ updateGroupInteraction - Updates a group interaction in DynamoDB
func (s *GroupInteractionService) updateGroupInteraction(ctx context.Context, interaction models.GroupInteraction) error {
	return s.Dynamo.PutItem(ctx, models.GroupInteractionsTable, interaction)
}

// ✅ createGroupInteractionForInvitee - Adds a new group record for an invitee
func (s *GroupInteractionService) createGroupInteractionForInvitee(ctx context.Context, invite models.GroupInteraction, groupId string) error {
	inviteForInvitee := models.GroupInteraction{
		PK:              "USER#" + invite.InviteeHandle,
		SK:              "GROUP#" + groupId,
		InteractionType: "group_chat",
		Status:          "active",
		GroupID:         &groupId,
		InviterHandle:   invite.InviterHandle,
		ApproverHandle:  invite.ApproverHandle,
		InviteeHandle:   invite.InviteeHandle,
		Members:         invite.Members,
		CreatedAt:       time.Now(),
		LastUpdated:     time.Now(),
	}

	return s.Dynamo.PutItem(ctx, models.GroupInteractionsTable, inviteForInvitee)
}

func contains(members []string, userHandle string) bool {
	for _, member := range members {
		if member == userHandle {
			return true
		}
	}
	return false
}

// ✅ createBatchGroupInteractions - Adds multiple group records in a single batch write
func (s *GroupInteractionService) createBatchGroupInteractions(ctx context.Context, groupRecords []models.GroupInteraction) error {
	var writeRequests []types.WriteRequest

	for _, record := range groupRecords {
		item, err := attributevalue.MarshalMap(record)
		if err != nil {
			log.Printf("❌ Error marshalling group interaction record: %v", err)
			return err
		}

		writeRequests = append(writeRequests, types.WriteRequest{
			PutRequest: &types.PutRequest{Item: item},
		})
	}

	err := s.Dynamo.BatchWriteItems(ctx, models.GroupInteractionsTable, writeRequests)
	if err != nil {
		log.Printf("❌ Error in batch write: %v", err)
		return err
	}

	log.Printf("✅ Successfully inserted %d group records", len(groupRecords))
	return nil
}
