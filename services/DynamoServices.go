package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
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

func (ds *DynamoService) PutItem(ctx context.Context, tableName string, item interface{}) error {
	log.Printf("Marshalling item for table '%s'...\n", tableName)
	marshaledItem, err := attributevalue.MarshalMap(item)
	if err != nil {
		log.Printf("Failed to marshal item: %v\n", err)
		return fmt.Errorf("failed to marshal item: %w", err)
	}
	log.Printf("Item marshalled: %+v\n", marshaledItem)

	log.Printf("Inserting item into table '%s'...\n", tableName)
	_, err = ds.Client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &tableName,
		Item:      marshaledItem,
	})
	if err != nil {
		log.Printf("Failed to insert item: %v\n", err)
		return fmt.Errorf("failed to put item in table '%s': %w", tableName, err)
	}
	log.Println("Item successfully inserted.")
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
func (ds *DynamoService) UpdateItem(
	ctx context.Context,
	tableName string,
	updateExpression string,
	key map[string]types.AttributeValue,
	expressionAttributeValues map[string]types.AttributeValue,
	expressionAttributeNames map[string]string,
) (map[string]types.AttributeValue, error) {

	log.Printf("🔄 Starting UpdateItem for table: %s", tableName)
	log.Printf("📝 Update Expression: %s", updateExpression)
	log.Printf("🔑 Key: %+v", key)
	log.Printf("📌 ExpressionAttributeValues: %+v", expressionAttributeValues)
	log.Printf("🏷️ ExpressionAttributeNames: %+v", expressionAttributeNames)

	// Ensure key is not empty
	if len(key) == 0 {
		log.Println("❌ Update failed: key cannot be empty")
		return nil, errors.New("update failed: key cannot be empty")
	}

	// Ensure updateExpression is not empty
	if updateExpression == "" {
		log.Println("❌ Update failed: updateExpression cannot be empty")
		return nil, errors.New("update failed: updateExpression cannot be empty")
	}

	// 🚀 Dynamically handle `REMOVE` expressions
	isRemoveOperation := len(expressionAttributeValues) == 0 &&
		(len(expressionAttributeNames) > 0 || updateExpression[:6] == "REMOVE")

	if len(expressionAttributeValues) == 0 && !isRemoveOperation {
		log.Println("❌ Update failed: expressionAttributeValues cannot be empty (except for REMOVE)")
		return nil, errors.New("update failed: expressionAttributeValues cannot be empty")
	}

	// Ensure `expressionAttributeValues` is nil if not required
	var expAttrValues map[string]types.AttributeValue
	if len(expressionAttributeValues) > 0 {
		expAttrValues = expressionAttributeValues
	} else {
		expAttrValues = nil // Set to nil for REMOVE expressions
	}

	// Construct the update input
	updateInput := &dynamodb.UpdateItemInput{
		TableName:                 &tableName,
		Key:                       key,
		UpdateExpression:          &updateExpression,
		ExpressionAttributeValues: expAttrValues,
		ExpressionAttributeNames:  expressionAttributeNames,
		ReturnValues:              types.ReturnValueAllNew,
	}

	log.Printf("🚀 Executing UpdateItem for table '%s' with input: %+v", tableName, updateInput)

	// Execute the update operation
	output, err := ds.Client.UpdateItem(ctx, updateInput)
	if err != nil {
		log.Printf("❌ Failed to update item in table '%s': %v", tableName, err)
		return nil, fmt.Errorf("failed to update item in table '%s': %w", tableName, err)
	}

	// ✅ Ensure attributes are returned
	if output.Attributes == nil {
		log.Printf("⚠️ Update executed, but no attributes were returned for table '%s'", tableName)
		return map[string]types.AttributeValue{}, nil // Return empty map instead of nil
	}

	log.Printf("✅ Successfully updated item in table '%s', Updated Attributes: %+v", tableName, output.Attributes)
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

// QueryItemsWithIndex queries items from DynamoDB using a Global Secondary Index (GSI)
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
	output, err := ds.Client.Query(ctx, &dynamodb.QueryInput{
		TableName:                 &tableName,
		IndexName:                 &indexName, // ✅ Specify GSI name
		KeyConditionExpression:    &keyConditionExpression,
		ExpressionAttributeValues: expressionAttributeValues,
		ExpressionAttributeNames:  expressionAttributeNames,
		Limit:                     &limit,
	})
	if err != nil {
		log.Printf("❌ Error querying GSI: %v", err)
		return nil, fmt.Errorf("failed to query GSI '%s': %w", indexName, err)
	}
	log.Printf("✅ Query successful. Retrieved %d items.", len(output.Items))
	return output.Items, nil
}

// QueryItemsWithOptions queries DynamoDB with sorting and limit options
func (ds *DynamoService) QueryItemsWithOptions(
	ctx context.Context,
	tableName string,
	keyConditionExpression string,
	expressionAttributeValues map[string]types.AttributeValue,
	expressionAttributeNames map[string]string,
	limit int32,
	latestFirst bool, // ✅ true = latest messages first, false = oldest messages first
) ([]map[string]types.AttributeValue, error) {
	log.Printf("🔍 Querying table '%s' with sorting: %v, limit: %d", tableName, latestFirst, limit)

	// ✅ Set ScanIndexForward: false (latest messages first)
	scanIndexForward := latestFirst == false // false = descending order (latest first)

	queryInput := &dynamodb.QueryInput{
		TableName:                 &tableName,
		KeyConditionExpression:    &keyConditionExpression,
		ExpressionAttributeValues: expressionAttributeValues,
		ExpressionAttributeNames:  expressionAttributeNames,
		Limit:                     &limit,
		ScanIndexForward:          &scanIndexForward, // ✅ Sorting applied
	}

	// ✅ Execute query
	output, err := ds.Client.Query(ctx, queryInput)
	if err != nil {
		log.Printf("❌ Failed to query DynamoDB table '%s': %v", tableName, err)
		return nil, fmt.Errorf("failed to query table '%s': %w", tableName, err)
	}

	log.Printf("✅ Retrieved %d items from table '%s'", len(output.Items), tableName)
	return output.Items, nil
}

// QueryItemsWithFilters queries items with both KeyConditionExpression and FilterExpression
func (s *DynamoService) QueryItemsWithFilters(
	ctx context.Context,
	tableName string,
	keyCondition string,
	expressionValues map[string]types.AttributeValue,
	expressionNames map[string]string,
	filterExpression string, // ✅ Added filterExpression as a parameter
) ([]map[string]types.AttributeValue, error) {

	// Define the QueryInput
	input := &dynamodb.QueryInput{
		TableName:                 aws.String(tableName),
		KeyConditionExpression:    aws.String(keyCondition),
		ExpressionAttributeValues: expressionValues,
	}

	// Add ExpressionAttributeNames if provided
	if len(expressionNames) > 0 {
		input.ExpressionAttributeNames = expressionNames
	}

	// ✅ Use filterExpression only if provided
	if filterExpression != "" {
		input.FilterExpression = aws.String(filterExpression)
	}

	// Execute query
	result, err := s.Client.Query(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to query items: %w", err)
	}

	return result.Items, nil
}
