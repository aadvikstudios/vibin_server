package socket

import (
	"log"

	socketio "github.com/googollee/go-socket.io"
)

// NewSocketServer initializes and returns a new Socket.IO server
func NewSocketServer() *socketio.Server {
	server := socketio.NewServer(nil)

	server.OnConnect("/", func(s socketio.Conn) error {
		log.Println("✅ Socket connected:", s.ID())
		return nil
	})

	server.OnEvent("/", "join", func(s socketio.Conn, data map[string]string) {
		matchID := data["matchId"]
		log.Printf("👥 User %s joined match %s\n", s.ID(), matchID)
		s.Join(matchID)
	})

	server.OnEvent("/", "sendMessage", func(s socketio.Conn, message map[string]interface{}) {
		matchID, ok := message["matchId"].(string)
		if !ok {
			log.Println("❌ Invalid matchId in message")
			return
		}
		log.Printf("📩 New message for match %s: %v\n", matchID, message)
		server.BroadcastToRoom("/", matchID, "newMessage", message)
	})

	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		log.Printf("❌ Socket disconnected: %s, Reason: %s\n", s.ID(), reason)
	})

	server.OnError("/", func(s socketio.Conn, err error) {
		log.Printf("⚠️ Socket error: %v", err)
	})

	return server
}
