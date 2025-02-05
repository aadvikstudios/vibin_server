package main

import (
	"encoding/json"
	"fmt"
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

	// Initialize Services
	userProfileService := &services.UserProfileService{Dynamo: dynamoService}
	actionService := &services.ActionService{Dynamo: dynamoService}
	chatService := &services.ChatService{Dynamo: dynamoService}
	matchService := &services.MatchService{Dynamo: dynamoService}

	// Set up the server port
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Using server port: %s\n", port)

	// Initialize the router
	r := mux.NewRouter()

	// Register a welcome route
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Welcome to Vibin")
	}).Methods("GET")

	// Register a health check endpoint
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := map[string]string{"status": "healthy"}
		json.NewEncoder(w).Encode(response)
	}).Methods("GET")

	// Register routes
	routes.RegisterUserProfileRoutes(r, userProfileService)
	routes.RegisterActionRoutes(r, actionService)
	routes.RegisterChatRoutes(r, chatService)
	routes.RegisterMatchRoutes(r, matchService)
	routes.RegisterS3Routes(r)

	// Add CORS middleware
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"}, // Adjust for specific domains if needed
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}).Handler(r)

	// Start the HTTP server
	log.Printf("Starting server on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, corsHandler))
}
