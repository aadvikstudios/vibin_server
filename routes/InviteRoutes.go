package routes

import (
	"vibin_server/controllers"
	"vibin_server/services"

	"github.com/gorilla/mux"
)

// RegisterPendingInviteRoutes registers all invite-related routes under `/api/invites`
func RegisterPendingInviteRoutes(router *mux.Router, inviteService *services.InviteService) {
	controller := &controllers.InviteController{InviteService: inviteService}

	inviteRouter := router.PathPrefix("/api/invites").Subrouter()
	inviteRouter.HandleFunc("", controller.CreateInviteHandler).Methods("POST")                          // Create an invite
	inviteRouter.HandleFunc("/pending/{approverId}", controller.GetPendingInvitesHandler).Methods("GET") // Get pending invites
	inviteRouter.HandleFunc("/sent/{inviterId}", controller.GetSentInvitesHandler).Methods("GET")        // Get sent invites
	inviteRouter.HandleFunc("/update", controller.UpdateInviteStatusHandler).Methods("PUT")              // Update invite status
}
