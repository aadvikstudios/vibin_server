package utils

import (
	"log"
	"strconv"

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

// ExtractInt safely extracts an integer from DynamoDB attribute map
func ExtractInt(profile map[string]types.AttributeValue, field string) int {
	if attr, ok := profile[field]; ok {
		if v, ok := attr.(*types.AttributeValueMemberN); ok {
			age, err := strconv.Atoi(v.Value) // Convert string to int
			if err == nil {
				return age
			}
		}
	}
	return 0 // Return 0 if conversion fails or field is missing
}

// ExtractBool safely extracts a boolean value from DynamoDB attribute map
func ExtractBool(profile map[string]types.AttributeValue, field string) bool {
	if attr, ok := profile[field]; ok {
		switch v := attr.(type) {
		case *types.AttributeValueMemberBOOL:
			return v.Value
		case *types.AttributeValueMemberS:
			// Handle cases where the boolean is stored as a string
			return v.Value == "true"
		case *types.AttributeValueMemberN:
			// Handle cases where the boolean is stored as a number (1 for true, 0 for false)
			return v.Value == "1"
		}
	}
	return false // Default to false if the field is missing or not a valid boolean
}

// ExtractFirstPhoto extracts the first photo URL from the "photos" attribute
func ExtractFirstPhoto(profile map[string]types.AttributeValue, field string) string {
	log.Println("ExtractFirstPhoto called with field:", field)
	if attr, ok := profile[field]; ok {
		if photos, ok := attr.(*types.AttributeValueMemberL); ok && len(photos.Value) > 0 {
			if photo, ok := photos.Value[0].(*types.AttributeValueMemberS); ok {
				return photo.Value
			}
		}
	}
	return ""
}

// ExtractPhotoURLs extracts photo URLs from a DynamoDB attribute
func ExtractPhotoURLs(profile map[string]types.AttributeValue) []string {
	photoURLs := []string{}
	if photosAttr, ok := profile["photos"]; ok {
		if photos, ok := photosAttr.(*types.AttributeValueMemberL); ok {
			for _, photo := range photos.Value {
				if photoURL, ok := photo.(*types.AttributeValueMemberS); ok {
					photoURLs = append(photoURLs, photoURL.Value)
				}
			}
		}
	}
	return photoURLs
}
