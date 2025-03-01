package routes

import (
	"vibin_server/controllers"
	"vibin_server/services"

	"github.com/gorilla/mux"
)

// RegisterGroupInteractionRoutes registers group interaction-related routes
func RegisterGroupInteractionRoutes(r *mux.Router, groupInteractionService *services.GroupInteractionService) {
	controller := controllers.NewGroupInteractionController(groupInteractionService)

	groupRouter := r.PathPrefix("/api/groupinteractions").Subrouter()

	// ✅ Create group invite (User A invites User C)
	groupRouter.HandleFunc("/invite", controller.CreateGroupInvite).Methods("POST")

	// ✅ Fetch invites created by User A (to check status)
	groupRouter.HandleFunc("/sent", controller.GetSentInvites).Methods("GET")

	// ✅ Fetch pending approvals for User B
	groupRouter.HandleFunc("/pending", controller.GetPendingApprovals).Methods("GET")

	// ✅ Approve or decline an invite
	groupRouter.HandleFunc("/approve", controller.ApproveOrDeclineInvite).Methods("POST")
	groupRouter.HandleFunc("/active/{userHandle}", controller.GetActiveGroups).Methods("GET")
}
