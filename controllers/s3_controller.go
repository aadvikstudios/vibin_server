package controllers

import (
	"encoding/json"
	"net/http"
	"vibin_server/services"
)

// GeneratePresignedURL generates a presigned URL for S3 uploads
func GeneratePresignedURL(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		FileName string `json:"fileName"`
		FileType string `json:"fileType"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.FileName == "" || payload.FileType == "" {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	url, fileName, err := services.GenerateUploadURL(payload.FileName, payload.FileType)
	if err != nil {
		http.Error(w, "Failed to generate pre-signed URL", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"url": url, "fileName": fileName})
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
