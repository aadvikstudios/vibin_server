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

	// Set up the server port
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize the router
	r := mux.NewRouter()

	// Pass DynamoService to routes for dependency injection
	routes.RegisterUserProfileRoutes(r, dynamoService)

	log.Printf("Server running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
