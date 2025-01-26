package routes

import (
	"vibin_server/controllers"

	"github.com/gorilla/mux"
)

// RegisterS3Routes sets up routes for S3-related operations
func RegisterS3Routes(r *mux.Router) {
	r.HandleFunc("/generate-presigned-url", controllers.GeneratePresignedURL).Methods("POST")
	r.HandleFunc("/get-presigned-read-url", controllers.GetPresignedReadURL).Methods("POST")
}
