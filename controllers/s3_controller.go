package controllers

import (
	"encoding/json"
	"log"
	"net/http"
	"vibin_server/services"
)

// GeneratePresignedURL generates a presigned URL for S3 uploads
func GeneratePresignedURL(w http.ResponseWriter, r *http.Request) {
	log.Println("GeneratePresignedURL: Received request")

	var payload struct {
		FileName string `json:"fileName"`
		FileType string `json:"fileType"`
	}

	// Decode JSON payload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Printf("Error decoding request body: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if payload.FileName == "" || payload.FileType == "" {
		log.Println("Error: Missing required fields in request payload")
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	log.Printf("GeneratePresignedURL: Generating pre-signed URL for FileName: %s, FileType: %s", payload.FileName, payload.FileType)

	url, fileName, err := services.GenerateUploadURL(payload.FileName, payload.FileType)
	if err != nil {
		log.Printf("Error generating pre-signed URL: %v", err)
		http.Error(w, "Failed to generate pre-signed URL", http.StatusInternalServerError)
		return
	}

	log.Printf("GeneratePresignedURL: Successfully generated URL: %s for file: %s", url, fileName)

	response := map[string]string{"url": url, "fileName": fileName}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	log.Println("GeneratePresignedURL: Response successfully sent")
}

// GetPresignedReadURL generates a presigned URL for reading S3 objects
func GetPresignedReadURL(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.Key == "" {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	url, err := services.GenerateReadURL(payload.Key)
	if err != nil {
		http.Error(w, "Failed to generate read pre-signed URL", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"url": url})
}
