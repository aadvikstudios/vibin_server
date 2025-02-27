package models

// ✅ Interaction Types (like, dislike, ping, invite, etc.)
const (
	InteractionTypeLike    = "like"
	InteractionTypeDislike = "dislike"
	InteractionTypePing    = "ping"
	InteractionTypeInvite  = "invite"
)

// ✅ Chat Types (private, group)
const (
	ChatTypePrivate = "private"
	ChatTypeGroup   = "group"
)

// ✅ Interaction Statuses
const (
	StatusPending  = "pending"
	StatusMatch    = "match"
	StatusSeen     = "seen"
	StatusDeclined = "declined"
	StatusApproved = "approved"
	StatusRejected = "rejected"
)
