package utils

import (
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// ExtractString safely extracts a string from a DynamoDB attribute map
func ExtractString(profile map[string]types.AttributeValue, field string) string {
	if attr, ok := profile[field]; ok {
		if v, ok := attr.(*types.AttributeValueMemberS); ok {
			return v.Value
		}
	}
	return ""
}

// ExtractFirstPhoto extracts the first photo URL from the "photos" attribute
func ExtractFirstPhoto(profile map[string]types.AttributeValue, field string) string {
	if attr, ok := profile[field]; ok {
		if photos, ok := attr.(*types.AttributeValueMemberL); ok && len(photos.Value) > 0 {
			if photo, ok := photos.Value[0].(*types.AttributeValueMemberS); ok {
				return photo.Value
			}
		}
	}
	return ""
}
