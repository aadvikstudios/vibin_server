package models

// UserProfile defines the structure for user profiles
type UserProfile struct {
	EmailID             string   `dynamodbav:"emailId" json:"emailId"`                                             // Partition Key
	EmailIDVerified     bool     `dynamodbav:"emailIdVerified,omitempty" json:"emailIdVerified,omitempty"`         // Email verification status
	PhoneNumber         string   `dynamodbav:"phoneNumber,omitempty" json:"phoneNumber,omitempty"`                 // User's phone number
	Name                string   `dynamodbav:"name,omitempty" json:"name,omitempty"`                               // Full name of the user
	UserID              string   `dynamodbav:"userId,omitempty" json:"userId,omitempty"`                           // Optional unique user ID
	Bio                 string   `dynamodbav:"bio,omitempty" json:"bio,omitempty"`                                 // Short biography
	Desires             []string `dynamodbav:"desires,omitempty" json:"desires,omitempty"`                         // User's desires
	DOB                 string   `dynamodbav:"dob,omitempty" json:"dob,omitempty"`                                 // Date of Birth
	Age                 int      `json:"age,omitempty"`                                                            // Calculated age
	Gender              string   `dynamodbav:"gender,omitempty" json:"gender,omitempty"`                           // Gender
	Interests           []string `dynamodbav:"interests,omitempty" json:"interests,omitempty"`                     // User's interests
	Latitude            float64  `dynamodbav:"latitude,omitempty" json:"latitude,omitempty"`                       // Latitude of the user's location
	Longitude           float64  `dynamodbav:"longitude,omitempty" json:"longitude,omitempty"`                     // Longitude of the user's location
	LookingFor          string   `dynamodbav:"lookingFor,omitempty" json:"lookingFor,omitempty"`                   // What the user is looking for
	Orientation         string   `dynamodbav:"orientation,omitempty" json:"orientation,omitempty"`                 // User's orientation
	ShowGenderOnProfile bool     `dynamodbav:"showGenderOnProfile,omitempty" json:"showGenderOnProfile,omitempty"` // Show gender on profile or not
	CountryCode         string   `json:"countryCode,omitempty"`                                                    // Country code
	Photos              []string `json:"photos,omitempty"`                                                         // User's photo URLs
}

// UserProfilesTable is the DynamoDB table name for user profiles
const UserProfilesTable = "UserProfiles"
