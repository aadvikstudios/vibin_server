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

// **1️⃣ Create an Invite (Method)**
func (s *InviteService) CreateInvite(ctx context.Context, inviterID, invitedUserID, approverID, matchID string) error {
	createdAt := time.Now().UTC().Format(time.RFC3339)

	invite := models.PendingInvite{
		ApproverID:    approverID,
		CreatedAt:     createdAt,
		InviterID:     inviterID,
		InvitedUserID: invitedUserID,
		MatchID:       matchID,
		Status:        models.InviteStatusPending, // ✅ Using constant
	}

	return s.Dynamo.PutItem(ctx, models.PendingInvite{}.TableName(), invite) // ✅ Using TableName()
}

// **2️⃣ Get Pending Invites (Method)**
func (s *InviteService) GetPendingInvites(ctx context.Context, approverID string) ([]models.PendingInvite, error) {
	tableName := models.PendingInvite{}.TableName()
	input := &dynamodb.QueryInput{
		TableName:              &tableName, // ✅ Assign to a variable first
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

// **3️⃣ Get Sent Invites (Method)**
func (s *InviteService) GetSentInvites(ctx context.Context, inviterID string) ([]models.PendingInvite, error) {
	tableName := models.PendingInvite{}.TableName()
	input := &dynamodb.QueryInput{ // ✅ Fix: Use `dynamodb.QueryInput`
		TableName:              &tableName, // ✅ Using TableName()
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

// **4️⃣ Accept/Decline an Invite (Method)**
func (s *InviteService) UpdateInviteStatus(ctx context.Context, approverID, createdAt, status string) error {
	// ✅ Using Constants for Status Validation
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

	_, err := s.Dynamo.UpdateItem(ctx, models.PendingInvite{}.TableName(), updateExpression, key, expressionValues, expressionNames) // ✅ Using TableName()
	return err
}
