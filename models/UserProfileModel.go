package models

// UserProfile defines the structure for user profiles
type UserProfile struct {
	UserHandle          string            `dynamodbav:"userhandle" json:"userhandle"`                                       // ✅ Partition Key
	EmailID             string            `dynamodbav:"emailId,omitempty" json:"emailId,omitempty"`                         // Indexed via GSI
	EmailIDVerified     bool              `dynamodbav:"emailIdVerified,omitempty" json:"emailIdVerified,omitempty"`         // Email verification status
	PhoneNumber         string            `dynamodbav:"phoneNumber,omitempty" json:"phoneNumber,omitempty"`                 // User's phone number
	Name                string            `dynamodbav:"name,omitempty" json:"name,omitempty"`                               // Full name of the user
	UserName            string            `dynamodbav:"username,omitempty" json:"username,omitempty"`                       // Display name
	HideName            bool              `dynamodbav:"hideName,omitempty" json:"hideName,omitempty"`                       // Flag to hide real name on profile
	Bio                 string            `dynamodbav:"bio,omitempty" json:"bio,omitempty"`                                 // Short biography
	Desires             []string          `dynamodbav:"desires,omitempty" json:"desires,omitempty"`                         // User's desires
	DOB                 string            `dynamodbav:"dob,omitempty" json:"dob,omitempty"`                                 // Date of Birth
	Age                 int               `dynamodbav:"age,omitempty" json:"age,omitempty"`                                 // Calculated age
	Gender              string            `dynamodbav:"gender,omitempty" json:"gender,omitempty"`                           // Gender
	Interests           []string          `dynamodbav:"interests,omitempty" json:"interests,omitempty"`                     // User's interests
	Latitude            float64           `dynamodbav:"latitude,omitempty" json:"latitude,omitempty"`                       // Latitude of the user's location
	Longitude           float64           `dynamodbav:"longitude,omitempty" json:"longitude,omitempty"`                     // Longitude of the user's location
	LookingFor          string            `dynamodbav:"lookingFor,omitempty" json:"lookingFor,omitempty"`                   // What the user is looking for
	Orientation         string            `dynamodbav:"orientation,omitempty" json:"orientation,omitempty"`                 // User's orientation
	ShowGenderOnProfile bool              `dynamodbav:"showGenderOnProfile,omitempty" json:"showGenderOnProfile,omitempty"` // Show gender on profile or not
	Photos              []string          `dynamodbav:"photos,omitempty" json:"photos,omitempty"`                           // User photos
	DistanceBetween     float64           `json:"distanceBetween" dynamodbav:"-"`                                           // Computed distance (not stored in DB)
	Questionnaire       map[string]string `dynamodbav:"questionnaire,omitempty" json:"questionnaire,omitempty"`             // Questionnaire responses
}

// UserProfilesTable is the DynamoDB table name for user profiles
const UserProfilesTable = "Users"
