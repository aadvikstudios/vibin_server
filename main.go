func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize the router
	r := mux.NewRouter()

	// Health Check Endpoint
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message": "Server is running!"}`))
	}).Methods("GET")

	// Welcome Endpoint
	r.HandleFunc("/welcome", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message": "Welcome to the server! This is the Vibin API."}`))
	}).Methods("GET")

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message": "Welcome to the server! This is the Vibin API."}`))
	}).Methods("GET")

	// Add Message Endpoint
	r.HandleFunc("/messages", func(w http.ResponseWriter, r *http.Request) {
		// Mock processing
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "Message added successfully", "data": {"messageId": "12345"}}`))
	}).Methods("POST")

	// Get Messages Endpoint
	r.HandleFunc("/messages/{matchId}", func(w http.ResponseWriter, r *http.Request) {
		matchID := mux.Vars(r)["matchId"]
		// Mock data
		response := `{"message": "Messages fetched successfully", "data": [{"matchId": "` + matchID + `", "content": "Hello, World!"}]}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}).Methods("GET")

	// Add User Profile Endpoint
	r.HandleFunc("/profiles", func(w http.ResponseWriter, r *http.Request) {
		// Mock processing
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "Profile added successfully", "data": {"userId": "abc123"}}`))
	}).Methods("POST")

	// Get User Profile Endpoint
	r.HandleFunc("/profiles/{userId}", func(w http.ResponseWriter, r *http.Request) {
		userID := mux.Vars(r)["userId"]
		// Mock data
		response := `{"message": "Profile fetched successfully", "data": {"userId": "` + userID + `", "fullName": "John Doe"}}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}).Methods("GET")

	// Delete User Profile Endpoint
	r.HandleFunc("/profiles/{userId}", func(w http.ResponseWriter, r *http.Request) {
		userID := mux.Vars(r)["userId"]
		// Mock processing
		response := `{"message": "Profile deleted successfully", "userId": "` + userID + `"}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}).Methods("DELETE")

	// Start the server
	log.Printf("Server running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
