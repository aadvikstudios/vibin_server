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
	Dynamo *DynamoService
}

// ✅ CreateGroupInvite - Adds a new group invite to DynamoDB
func (s *GroupInteractionService) CreateGroupInvite(ctx context.Context, invite models.GroupInteraction) error {
	return s.Dynamo.PutItem(ctx, models.GroupInteractionsTable, invite)
}

// ✅ GetSentInvites - Fetches invites created by User A
func (s *GroupInteractionService) GetSentInvites(ctx context.Context, userHandle string) ([]models.GroupInteraction, error) {
	keyCondition := "PK = :pk"
	expressionValues := map[string]types.AttributeValue{
		":pk": &types.AttributeValueMemberS{Value: "USER#" + userHandle},
	}

	items, err := s.Dynamo.QueryItems(ctx, models.GroupInteractionsTable, keyCondition, expressionValues, nil, 0)
	if err != nil {
		return nil, err
	}

	var invites []models.GroupInteraction
	if err := attributevalue.UnmarshalListOfMaps(items, &invites); err != nil {
		return nil, err
	}

	return invites, nil
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

	// Fetch the invite entry
	key := map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: "USER#" + approverHandle},
		"SK": &types.AttributeValueMemberS{Value: "PENDING_APPROVAL#GROUP_INVITE#" + inviteeHandle},
	}

	item, err := s.Dynamo.GetItem(ctx, models.GroupInteractionsTable, key)
	if err != nil {
		return err
	}

	if item == nil {
		return errors.New("invite not found")
	}

	var invite models.GroupInteraction
	if err := attributevalue.UnmarshalMap(item, &invite); err != nil {
		return err
	}

	// If the invite is approved, generate a group ID
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

	// Update item in DynamoDB
	if err := s.Dynamo.PutItem(ctx, models.GroupInteractionsTable, invite); err != nil {
		return err
	}

	// If approved, add the group interaction for the invitee
	if status == "approved" {
		inviteForInvitee := models.GroupInteraction{
			PK:              "USER#" + invite.InviteeHandle,
			SK:              "GROUP#" + *groupId,
			InteractionType: "group_chat",
			Status:          "active",
			GroupID:         groupId,
			InviterHandle:   invite.InviterHandle,
			ApproverHandle:  invite.ApproverHandle,
			InviteeHandle:   invite.InviteeHandle,
			Members:         invite.Members,
			CreatedAt:       time.Now(),
			LastUpdated:     time.Now(),
		}

		// Put the new group interaction for the invitee
		if err := s.Dynamo.PutItem(ctx, models.GroupInteractionsTable, inviteForInvitee); err != nil {
			return err
		}
	}

	return nil
}
