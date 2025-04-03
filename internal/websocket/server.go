// server.go
package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	_ "websocket-messaging/internal/database"
	"websocket-messaging/internal/redis"
)

type WebSocketServer struct {
	serverID string
	clients  map[string]*websocket.Conn
	redis    *redis.RedisClient
	upgrader websocket.Upgrader
}

func NewWebSocketServer(serverID string, redis *redis.RedisClient) *WebSocketServer {
	return &WebSocketServer{
		serverID: serverID,
		clients:  make(map[string]*websocket.Conn),
		redis:    redis,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (ws *WebSocketServer) HandleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade failed:", err)
		return
	}

	userID := r.URL.Query().Get("userID")
	ws.clients[userID] = conn
	ws.redis.SetUserServer(userID, ws.serverID)

	go ws.readMessages(userID, conn)
}

func (ws *WebSocketServer) readMessages(userID string, conn *websocket.Conn) {
	defer conn.Close()
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading message:", err)
			delete(ws.clients, userID)
			return
		}
		log.Printf("Received message from %s: %s", userID, string(message))
	}
}

func (ws *WebSocketServer) SendMessage(userID, message string) {
	if conn, exists := ws.clients[userID]; exists {
		err := conn.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			log.Println("Error sending message:", err)
		}
	}
}

func (ws *WebSocketServer) Start(port string) {
	http.HandleFunc("/ws", ws.HandleConnection)
	log.Println("WebSocket Server running on port", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func (ws *WebSocketServer) StartHeartbeat() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		ws.redis.SetWithTTL("server_heartbeat:"+ws.serverID, "alive", 10*time.Second)
	}
}

type MessagePayload struct {
	UserID  string
	Message string
}

func (ws *WebSocketServer) StartRedisListener() {
	channel := "ws_channel:" + ws.serverID
	ws.redis.Subscribe(channel, func(message string) {
		var msgData MessagePayload
		json.Unmarshal([]byte(message), &msgData)
		ws.SendMessage(msgData.UserID, msgData.Message)
	})
}
