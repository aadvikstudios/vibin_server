package socket

import (
	"log"

	gosocketio "github.com/erock530/gosf-socketio"
)

// NewSocketServer initializes and returns a new Socket.IO server
func NewSocketServer() *gosocketio.Server {
	server := gosocketio.NewServer(nil)

	// Handle connection events
	server.On(gosocketio.OnConnection, func(c *gosocketio.Channel) {
		log.Println("✅ Socket connected:", c.Id())
	})

	// Handle join events
	server.On("join", func(c *gosocketio.Channel, data map[string]string) {
		matchID := data["matchId"]
		if matchID == "" {
			log.Println("❌ Invalid matchId in join request")
			return
		}
		log.Printf("👥 User %s joined match %s\n", c.Id(), matchID)
		c.Join(matchID)
	})

	// Handle sendMessage events
	server.On("sendMessage", func(c *gosocketio.Channel, message map[string]interface{}) {
		matchID := message["matchId"].(string)
		log.Printf("📩 New message for match %s: %v\n", matchID, message)
		server.BroadcastTo(matchID, "newMessage", message)
	})

	// Handle disconnection
	server.On(gosocketio.OnDisconnection, func(c *gosocketio.Channel) {
		log.Println("❌ Socket disconnected:", c.Id())
	})

	return server
}
