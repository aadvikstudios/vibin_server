package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoService struct {
	Client *dynamodb.Client
}

// InitializeDynamoDBClient initializes the DynamoDB client
func InitializeDynamoDBClient() *dynamodb.Client {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(os.Getenv("AWS_REGION")))
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}
	return dynamodb.NewFromConfig(cfg)
}

// QueryItems queries items from DynamoDB using a KeyConditionExpression
func (ds *DynamoService) QueryItems(
	ctx context.Context,
	tableName string,
	keyConditionExpression string,
	expressionAttributeValues map[string]types.AttributeValue,
	expressionAttributeNames map[string]string,
	limit int32,
) ([]map[string]types.AttributeValue, error) {
	output, err := ds.Client.Query(ctx, &dynamodb.QueryInput{
		TableName:                 &tableName,
		KeyConditionExpression:    &keyConditionExpression,
		ExpressionAttributeValues: expressionAttributeValues,
		ExpressionAttributeNames:  expressionAttributeNames,
		Limit:                     &limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query items from table '%s': %w", tableName, err)
	}

	return output.Items, nil
}

// PutItem inserts an item into DynamoDB
func (ds *DynamoService) PutItem(ctx context.Context, tableName string, item interface{}) error {
	marshaledItem, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}

	_, err = ds.Client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &tableName,
		Item:      marshaledItem,
	})
	if err != nil {
		return fmt.Errorf("failed to put item in table '%s': %w", tableName, err)
	}
	return nil
}

// GetItem retrieves an item from DynamoDB
func (ds *DynamoService) GetItem(ctx context.Context, tableName string, key map[string]types.AttributeValue) (map[string]types.AttributeValue, error) {
	output, err := ds.Client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &tableName,
		Key:       key,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get item from table '%s': %w", tableName, err)
	}

	if output.Item == nil {
		return nil, errors.New("item not found")
	}

	return output.Item, nil
}

// UpdateItem updates an existing item in DynamoDB
func (ds *DynamoService) UpdateItem(
	ctx context.Context,
	tableName string,
	updateExpression string,
	key map[string]types.AttributeValue,
	expressionAttributeValues map[string]types.AttributeValue,
	expressionAttributeNames map[string]string,
) (map[string]types.AttributeValue, error) {
	output, err := ds.Client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 &tableName,
		Key:                       key,
		UpdateExpression:          &updateExpression,
		ExpressionAttributeValues: expressionAttributeValues,
		ExpressionAttributeNames:  expressionAttributeNames,
		ReturnValues:              types.ReturnValueAllNew,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update item in table '%s': %w", tableName, err)
	}

	return output.Attributes, nil
}

// DeleteItem removes an item from DynamoDB
func (ds *DynamoService) DeleteItem(ctx context.Context, tableName string, key map[string]types.AttributeValue) error {
	_, err := ds.Client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: &tableName,
		Key:       key,
	})
	if err != nil {
		return fmt.Errorf("failed to delete item from table '%s': %w", tableName, err)
	}
	return nil
}
