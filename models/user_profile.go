package models

// UserProfile defines the structure for user profiles
type UserProfile struct {
	UserID              string   `dynamodbav:"userId,omitempty"`
	FullName            string   `dynamodbav:"fullName,omitempty"`
	EmailID             string   `dynamodbav:"emailId,omitempty"`
	Bio                 string   `dynamodbav:"bio,omitempty"`
	Desires             []string `dynamodbav:"desires,omitempty"`
	DOB                 string   `dynamodbav:"dob,omitempty"`
	Gender              string   `dynamodbav:"gender,omitempty"`
	Interests           []string `dynamodbav:"interests,omitempty"`
	Latitude            float64  `dynamodbav:"latitude,omitempty"`
	Longitude           float64  `dynamodbav:"longitude,omitempty"`
	LookingFor          string   `dynamodbav:"lookingFor,omitempty"`
	Orientation         string   `dynamodbav:"orientation,omitempty"`
	ShowGenderOnProfile bool     `dynamodbav:"showGenderOnProfile,omitempty"`
}

// UserProfilesTable is the DynamoDB table name for user profiles
const UserProfilesTable = "UserProfiles"
