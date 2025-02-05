package main

import (
	"log"
	"net/http"
	"os"

	"vibin_server/socket" // Import the new socket package

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func main() {
	// Initialize DynamoDB client and services (your existing code)

	// Initialize Socket.IO server from the new package
	server := socket.NewSocketServer()
	go server.Serve()
	defer server.Close()

	// Set up the server port
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Using server port: %s\n", port)

	// Initialize the router
	r := mux.NewRouter()

	// Register your existing routes
	// routes.Register...

	// Combine Gorilla Mux and Socket.IO
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: false,
	}).Handler(r)

	http.Handle("/socket.io/", server)
	http.Handle("/", corsHandler)

	log.Printf("Starting server on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
