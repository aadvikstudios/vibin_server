package services

import (
	"context"
	"errors"
	"time"
	"vibin_server/models"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// InviteService handles operations related to pending invites
type InviteService struct {
	Dynamo *DynamoService
}

// **Create a New Invite**
func (s *InviteService) CreateInvite(ctx context.Context, inviterID, invitedUserID, approverID, inviteType, matchID string) error {
	createdAt := time.Now().UTC().Format(time.RFC3339)

	invite := models.PendingInvite{
		ApproverID:    approverID,
		CreatedAt:     createdAt,
		InviterID:     inviterID,
		InvitedUserID: invitedUserID,
		MatchID:       matchID,
		InviteType:    inviteType,
		Status:        models.InviteStatusPending,
	}

	return s.Dynamo.PutItem(ctx, models.PendingInvite{}.TableName(), invite)
}

// **Fetch Invite by Approver & Time (Required for Approval Process)**
func (s *InviteService) GetInviteByApproverAndTime(ctx context.Context, approverID, createdAt string) (*models.PendingInvite, error) {
	tableName := models.PendingInvite{}.TableName()
	key := map[string]types.AttributeValue{
		"approverId": &types.AttributeValueMemberS{Value: approverID},
		"createdAt":  &types.AttributeValueMemberS{Value: createdAt},
	}

	item, err := s.Dynamo.GetItem(ctx, tableName, key)
	if err != nil {
		return nil, err
	}

	var invite models.PendingInvite
	err = attributevalue.UnmarshalMap(item, &invite)
	if err != nil {
		return nil, err
	}

	return &invite, nil
}

// **Create a New Group Chat**
func (s *InviteService) CreateGroupMatch(ctx context.Context, matchID string, users []string) error {
	groupChat := models.Match{
		MatchID:   matchID,
		Users:     users,
		Type:      "group",
		Status:    "active",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	return s.Dynamo.PutItem(ctx, models.MatchesTable, groupChat)
}

// **Update Invite Status (Accept/Decline)**
func (s *InviteService) UpdateInviteStatus(ctx context.Context, approverID, createdAt, status string) error {
	if status != models.InviteStatusAccepted && status != models.InviteStatusDeclined {
		return errors.New("invalid status")
	}

	updateExpression := "SET #s = :status"
	key := map[string]types.AttributeValue{
		"approverId": &types.AttributeValueMemberS{Value: approverID},
		"createdAt":  &types.AttributeValueMemberS{Value: createdAt},
	}
	expressionValues := map[string]types.AttributeValue{
		":status": &types.AttributeValueMemberS{Value: status},
	}
	expressionNames := map[string]string{
		"#s": "status",
	}

	_, err := s.Dynamo.UpdateItem(ctx, models.PendingInvite{}.TableName(), updateExpression, key, expressionValues, expressionNames)
	return err
}

// **Fetch Pending Invites for Approver**
func (s *InviteService) GetPendingInvites(ctx context.Context, approverID string) ([]models.PendingInvite, error) {
	tableName := models.PendingInvite{}.TableName()
	input := &dynamodb.QueryInput{
		TableName:              &tableName,
		KeyConditionExpression: aws.String("approverId = :approverId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":approverId": &types.AttributeValueMemberS{Value: approverID},
		},
	}

	items, err := s.Dynamo.QueryItemsWithQueryInput(ctx, input)
	if err != nil {
		return nil, err
	}

	var invites []models.PendingInvite
	err = attributevalue.UnmarshalListOfMaps(items, &invites)
	return invites, err
}

// **Fetch Sent Invites for Inviter**
func (s *InviteService) GetSentInvites(ctx context.Context, inviterID string) ([]models.PendingInvite, error) {
	tableName := models.PendingInvite{}.TableName()
	input := &dynamodb.QueryInput{
		TableName:              &tableName,
		IndexName:              aws.String("InviterIndex"),
		KeyConditionExpression: aws.String("inviterId = :inviterId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":inviterId": &types.AttributeValueMemberS{Value: inviterID},
		},
	}

	items, err := s.Dynamo.QueryItemsWithQueryInput(ctx, input)
	if err != nil {
		return nil, err
	}

	var invites []models.PendingInvite
	err = attributevalue.UnmarshalListOfMaps(items, &invites)
	return invites, err
}
