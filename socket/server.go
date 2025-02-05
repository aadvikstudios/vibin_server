package socket

import (
	"log"

	socketio "github.com/googollee/go-socket.io"
)

// NewSocketServer initializes and returns a new Socket.IO server
func NewSocketServer() *socketio.Server {
	server := socketio.NewServer(nil)

	// Define Socket.IO event handlers
	server.OnConnect("/", func(s socketio.Conn) error {
		log.Println("Socket connected:", s.ID())
		return nil
	})

	server.OnEvent("/", "join", func(s socketio.Conn, data map[string]string) {
		matchID := data["matchId"]
		log.Printf("User %s joined match %s\n", s.ID(), matchID)
		s.Join(matchID)
	})

	server.OnEvent("/", "sendMessage", func(s socketio.Conn, message map[string]interface{}) {
		matchID := message["matchId"].(string)
		log.Printf("New message for match %s: %v\n", matchID, message)
		server.BroadcastToRoom("/", matchID, "newMessage", message)
	})

	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		log.Printf("Socket disconnected: %s, Reason: %s\n", s.ID(), reason)
	})

	return server
}
