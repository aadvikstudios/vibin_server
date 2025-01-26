package main

import (
	"log"
	"net/http"
	"os"

	"vibin_server/routes"
	"vibin_server/services"

	"github.com/gorilla/mux"
)

func main() {
	// Initialize DynamoDB client and service
	dynamoClient := services.InitializeDynamoDBClient()
	dynamoService := &services.DynamoService{Client: dynamoClient}

	// Initialize UserProfileService
	userProfileService := &services.UserProfileService{Dynamo: dynamoService}

	// Initialize ActionService (if needed)
	actionService := &services.ActionService{Dynamo: dynamoService}

	// Set up the server port
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize the router
	r := mux.NewRouter()

	// Register routes
	routes.RegisterUserProfileRoutes(r, userProfileService)
	routes.RegisterActionRoutes(r, actionService)

	// Start the server
	log.Printf("Server running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
