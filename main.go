package main

import (
	"log"
	"net/http"
	"os"

	"vibin_server/routes"
	"vibin_server/services"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func main() {
	// Initialize DynamoDB client and service
	log.Println("Initializing DynamoDB client...")
	dynamoClient := services.InitializeDynamoDBClient()
	dynamoService := &services.DynamoService{Client: dynamoClient}
	log.Println("DynamoDB client initialized.")

	// Initialize UserProfileService
	log.Println("Initializing UserProfileService...")
	userProfileService := &services.UserProfileService{Dynamo: dynamoService}
	log.Println("UserProfileService initialized.")

	// Initialize ActionService
	log.Println("Initializing ActionService...")
	actionService := &services.ActionService{Dynamo: dynamoService}
	log.Println("ActionService initialized.")

	// Initialize ChatService
	log.Println("Initializing ChatService...")
	chatService := &services.ChatService{Dynamo: dynamoService}
	log.Println("ChatService initialized.")

	// Initialize ChatService
	log.Println("Initializing ChatService...")
	matchService := &services.MatchService{Dynamo: dynamoService}
	log.Println("ChatService initialized.")

	// Set up the server port
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Using server port: %s\n", port)

	// Initialize the router
	log.Println("Initializing router...")
	r := mux.NewRouter()

	// Register routes
	log.Println("Registering routes...")
	routes.RegisterUserProfileRoutes(r, userProfileService)
	routes.RegisterActionRoutes(r, actionService)
	routes.RegisterChatRoutes(r, chatService)
	routes.RegisterMatchRoutes(r, matchService)
	routes.RegisterS3Routes(r)
	log.Println("Routes registered.")

	// Add CORS middleware
	log.Println("Configuring CORS...")
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: false,
	}).Handler(r)

	// Start the server
	log.Printf("Starting server on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, corsHandler))
}
