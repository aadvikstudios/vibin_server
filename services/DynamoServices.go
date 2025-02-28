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

func (d *DynamoService) QueryItemsWithQueryInput(ctx context.Context, input *dynamodb.QueryInput) ([]map[string]types.AttributeValue, error) {
	// Execute DynamoDB Query
	result, err := d.Client.Query(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to query DynamoDB: %w", err)
	}

	return result.Items, nil
}

func (ds *DynamoService) ScanWithFilter(
	ctx context.Context,
	tableName string,
	filterFunc func(map[string]types.AttributeValue) bool, // Callback for additional filtering
	excludeFields map[string]string, // Fields to exclude specific values
	result interface{}, // Pointer to a slice of structs to store results
) error {
	// Build FilterExpression
	var filterExpressions []string
	expressionAttributeNames := map[string]string{}
	expressionAttributeValues := map[string]types.AttributeValue{}

	// Exclude fields
	for key, value := range excludeFields {
		expressionAttributeNames["#"+key] = key
		expressionAttributeValues[":"+key] = &types.AttributeValueMemberS{Value: value}
		filterExpressions = append(filterExpressions, fmt.Sprintf("#%s <> :%s", key, key))
	}

	// Combine expressions
	filterExpression := ""
	if len(filterExpressions) > 0 {
		filterExpression = stringJoin(filterExpressions, " AND ")
	}

	// Perform a full scan of the DynamoDB table
	scanInput := &dynamodb.ScanInput{
		TableName:                 &tableName,
		FilterExpression:          &filterExpression,
		ExpressionAttributeNames:  expressionAttributeNames,
		ExpressionAttributeValues: expressionAttributeValues,
	}

	output, err := ds.Client.Scan(ctx, scanInput)
	if err != nil {
		return fmt.Errorf("failed to scan table '%s': %w", tableName, err)
	}

	// Apply the additional filtering callback if provided
	var filteredItems []map[string]types.AttributeValue
	for _, item := range output.Items {
		if filterFunc == nil || filterFunc(item) {
			filteredItems = append(filteredItems, item)
		}
	}

	// Unmarshal filtered items into the result
	if err := attributevalue.UnmarshalListOfMaps(filteredItems, result); err != nil {
		return fmt.Errorf("failed to unmarshal scan result: %w", err)
	}

	return nil
}

// Utility function to join strings
func stringJoin(parts []string, delimiter string) string {
	result := ""
	for i, part := range parts {
		if i > 0 {
			result += delimiter
		}
		result += part
	}
	return result
}

// BatchWriteItems writes multiple items to DynamoDB in batches
func (ds *DynamoService) BatchWriteItems(
	ctx context.Context,
	tableName string,
	writeRequests []types.WriteRequest,
) error {
	const maxBatchSize = 25

	// Process requests in batches of 25
	for i := 0; i < len(writeRequests); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(writeRequests) {
			end = len(writeRequests)
		}

		// Create batch input
		batchInput := &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{
				tableName: writeRequests[i:end],
			},
		}

		// Execute batch write
		_, err := ds.Client.BatchWriteItem(ctx, batchInput)
		if err != nil {
			return fmt.Errorf("failed to batch write items to table '%s': %w", tableName, err)
		}
	}

	return nil
}

// ✅ Query items from DynamoDB using a GSI with optional filters
func (ds *DynamoService) QueryItemsWithIndexWithFilters(
	ctx context.Context,
	tableName string,
	indexName string,
	keyConditionExpression string,
	expressionAttributeValues map[string]types.AttributeValue,
	expressionAttributeNames map[string]string,
	filterExpression string, // ✅ Optional filter expression
	limit int32,
) ([]map[string]types.AttributeValue, error) {
	log.Printf("🔍 Querying GSI: %s in table: %s", indexName, tableName)

	queryInput := &dynamodb.QueryInput{
		TableName:                 &tableName,
		IndexName:                 &indexName,
		KeyConditionExpression:    &keyConditionExpression,
		ExpressionAttributeValues: expressionAttributeValues,
		ExpressionAttributeNames:  expressionAttributeNames,
		Limit:                     &limit,
	}

	// ✅ Apply FilterExpression if provided
	if filterExpression != "" {
		queryInput.FilterExpression = &filterExpression
	}

	output, err := ds.Client.Query(ctx, queryInput)
	if err != nil {
		log.Printf("❌ Error querying GSI: %v", err)
		return nil, fmt.Errorf("failed to query GSI '%s': %w", indexName, err)
	}
	log.Printf("✅ Query successful. Retrieved %d items.", len(output.Items))
	return output.Items, nil
}

// ✅ Query items with only KeyConditionExpression (No filters)
func (ds *DynamoService) QueryItemsWithIndex(
	ctx context.Context,
	tableName string,
	indexName string,
	keyConditionExpression string,
	expressionAttributeValues map[string]types.AttributeValue,
	expressionAttributeNames map[string]string,
	limit int32,
) ([]map[string]types.AttributeValue, error) {
	log.Printf("🔍 Querying GSI: %s in table: %s", indexName, tableName)

	queryInput := &dynamodb.QueryInput{
		TableName:                 &tableName,
		IndexName:                 &indexName,
		KeyConditionExpression:    &keyConditionExpression,
		ExpressionAttributeValues: expressionAttributeValues,
		ExpressionAttributeNames:  expressionAttributeNames,
		Limit:                     &limit,
	}

	output, err := ds.Client.Query(ctx, queryInput)
	if err != nil {
		log.Printf("❌ Error querying GSI: %v", err)
		return nil, fmt.Errorf("failed to query GSI '%s': %w", indexName, err)
	}
	log.Printf("✅ Query successful. Retrieved %d items.", len(output.Items))
	return output.Items, nil
}

// ✅ Get item from DynamoDB
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

// ✅ Put item into DynamoDB
func (ds *DynamoService) PutItem(ctx context.Context, tableName string, item interface{}) error {
	log.Printf("📝 Marshalling item for table '%s'...", tableName)
	marshaledItem, err := attributevalue.MarshalMap(item)
	if err != nil {
		log.Printf("❌ Failed to marshal item: %v", err)
		return fmt.Errorf("failed to marshal item: %w", err)
	}

	log.Printf("🚀 Inserting item into table '%s'...", tableName)
	_, err = ds.Client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &tableName,
		Item:      marshaledItem,
	})
	if err != nil {
		log.Printf("❌ Failed to insert item: %v", err)
		return fmt.Errorf("failed to put item in table '%s': %w", tableName, err)
	}
	log.Println("✅ Item successfully inserted.")
	return nil
}

// ✅ Update item in DynamoDB
func (ds *DynamoService) UpdateItem(
	ctx context.Context,
	tableName string,
	updateExpression string,
	key map[string]types.AttributeValue,
	expressionAttributeValues map[string]types.AttributeValue,
	expressionAttributeNames map[string]string,
) (map[string]types.AttributeValue, error) {
	log.Printf("🔄 Updating item in table: %s", tableName)

	updateInput := &dynamodb.UpdateItemInput{
		TableName:                 &tableName,
		Key:                       key,
		UpdateExpression:          &updateExpression,
		ExpressionAttributeValues: expressionAttributeValues,
		ExpressionAttributeNames:  expressionAttributeNames,
		ReturnValues:              types.ReturnValueAllNew,
	}

	output, err := ds.Client.UpdateItem(ctx, updateInput)
	if err != nil {
		log.Printf("❌ Failed to update item in table '%s': %v", tableName, err)
		return nil, fmt.Errorf("failed to update item in table '%s': %w", tableName, err)
	}

	if output.Attributes == nil {
		log.Printf("⚠️ Update executed, but no attributes were returned for table '%s'", tableName)
		return map[string]types.AttributeValue{}, nil
	}

	log.Printf("✅ Successfully updated item in table '%s'", tableName)
	return output.Attributes, nil
}

// ✅ Delete item from DynamoDB
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

// ✅ Query items with sorting & limit options
func (ds *DynamoService) QueryItemsWithOptions(
	ctx context.Context,
	tableName string,
	keyConditionExpression string,
	expressionAttributeValues map[string]types.AttributeValue,
	expressionAttributeNames map[string]string,
	limit int32,
	latestFirst bool,
) ([]map[string]types.AttributeValue, error) {
	log.Printf("🔍 Querying table '%s' with sorting: %v, limit: %d", tableName, latestFirst, limit)

	scanIndexForward := latestFirst == false // `false` = latest first
	queryInput := &dynamodb.QueryInput{
		TableName:                 &tableName,
		KeyConditionExpression:    &keyConditionExpression,
		ExpressionAttributeValues: expressionAttributeValues,
		ExpressionAttributeNames:  expressionAttributeNames,
		Limit:                     &limit,
		ScanIndexForward:          &scanIndexForward,
	}

	output, err := ds.Client.Query(ctx, queryInput)
	if err != nil {
		log.Printf("❌ Failed to query DynamoDB table '%s': %v", tableName, err)
		return nil, fmt.Errorf("failed to query table '%s': %w", tableName, err)
	}

	log.Printf("✅ Retrieved %d items from table '%s'", len(output.Items), tableName)
	return output.Items, nil
}
