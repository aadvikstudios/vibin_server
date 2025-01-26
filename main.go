package main

import (
	"log"
	"net/http"
	"os"

	"vibin_server/routes"

	"github.com/gorilla/mux"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize the router and register routes
	r := mux.NewRouter()
	routes.RegisterRoutes(r)

	// Start the server
	log.Printf("Server running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
