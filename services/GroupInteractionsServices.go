package services

import (
	"context"
	"errors"
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
	// ✅ Step 1: Validate InviteeHandle (Check if user exists)
	profile, err := s.UserProfileService.GetUserProfileByHandle(ctx, invite.InviteeHandle)
	if err != nil {
		return errors.New("failed to validate invitee handle")
	}
	if profile == nil {
		return errors.New("invitee handle does not exist")
	}

	// ✅ Step 2: Store the invite in DynamoDB
	return s.Dynamo.PutItem(ctx, models.GroupInteractionsTable, invite)
}

// ✅ GetSentInvites - Fetches invites created by User A
func (s *GroupInteractionService) GetSentInvites(ctx context.Context, userHandle string) ([]models.GroupInteraction, error) {
	return s.queryGroupInteractions(ctx, "USER#"+userHandle)
}

// ✅ GetPendingApprovals - Fetches pending invites for User B
func (s *GroupInteractionService) GetPendingApprovals(ctx context.Context, approverHandle string) ([]models.GroupInteraction, error) {
	keyCondition := "approverHandle = :approver AND status = :status"
	expressionValues := map[string]types.AttributeValue{
		":approver": &types.AttributeValueMemberS{Value: approverHandle},
		":status":   &types.AttributeValueMemberS{Value: "pending"},
	}

	items, err := s.Dynamo.QueryItemsWithIndex(ctx, models.GroupInteractionsTable, models.ApprovalIndex, keyCondition, expressionValues, nil, 0)
	if err != nil {
		return nil, err
	}

	var pendingInvites []models.GroupInteraction
	if err := attributevalue.UnmarshalListOfMaps(items, &pendingInvites); err != nil {
		return nil, err
	}

	return pendingInvites, nil
}

// ✅ ApproveOrDeclineInvite - Approves or declines a pending invite
func (s *GroupInteractionService) ApproveOrDeclineInvite(ctx context.Context, approverHandle, inviteeHandle, status string) error {
	// Validate status
	if status != "approved" && status != "declined" {
		return errors.New("invalid status value")
	}

	// Fetch existing invite
	invite, err := s.getGroupInteraction(ctx, "USER#"+approverHandle, "PENDING_APPROVAL#GROUP_INVITE#"+inviteeHandle)
	if err != nil {
		return err
	}

	// If approved, generate a group ID
	var groupId *string
	if status == "approved" {
		newGroupId := uuid.New().String()
		groupId = &newGroupId
	}

	// Update the invite status
	invite.Status = status
	invite.GroupID = groupId
	invite.Members = append(invite.Members, invite.InviteeHandle) // Add invitee to members list
	invite.LastUpdated = time.Now()

	// Save updated invite
	if err := s.updateGroupInteraction(ctx, *invite); err != nil {
		return err
	}

	// If approved, add the group interaction for the invitee
	if status == "approved" {
		return s.createGroupInteractionForInvitee(ctx, *invite, *groupId)
	}

	return nil
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
