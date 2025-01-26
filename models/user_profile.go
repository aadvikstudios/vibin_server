package models

// UserProfile defines the structure for user profiles
type UserProfile struct {
	UserID              string   `dynamodbav:"userId,omitempty" json:"userId,omitempty"`
	Name                string   `dynamodbav:"name,omitempty" json:"name,omitempty"`
	EmailID             string   `dynamodbav:"emailId,omitempty" json:"emailId,omitempty"`
	EmailIDVerified     bool     `dynamodbav:"emailIdVerified,omitempty" json:"emailIdVerified,omitempty"`
	PhoneNumber         string   `dynamodbav:"phoneNumber,omitempty" json:"phoneNumber,omitempty"`
	Bio                 string   `dynamodbav:"bio,omitempty" json:"bio,omitempty"`
	Desires             []string `dynamodbav:"desires,omitempty" json:"desires,omitempty"`
	DOB                 string   `dynamodbav:"dob,omitempty" json:"dob,omitempty"`
	Age                 int      `json:"age,omitempty"`
	Gender              string   `dynamodbav:"gender,omitempty" json:"gender,omitempty"`
	Interests           []string `dynamodbav:"interests,omitempty" json:"interests,omitempty"`
	Latitude            float64  `dynamodbav:"latitude,omitempty" json:"latitude,omitempty"`
	Longitude           float64  `dynamodbav:"longitude,omitempty" json:"longitude,omitempty"`
	LookingFor          string   `dynamodbav:"lookingFor,omitempty" json:"lookingFor,omitempty"`
	Orientation         string   `dynamodbav:"orientation,omitempty" json:"orientation,omitempty"`
	ShowGenderOnProfile bool     `dynamodbav:"showGenderOnProfile,omitempty" json:"showGenderOnProfile,omitempty"`
	CountryCode         string   `json:"countryCode,omitempty"`
	Photos              []string `json:"photos,omitempty"`
}

// UserProfilesTable is the DynamoDB table name for user profiles
const UserProfilesTable = "UserProfiles"
