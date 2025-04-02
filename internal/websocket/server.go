// server.go
package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	_ "websocket-messaging/internal/database"
	"websocket-messaging/internal/models"
	"websocket-messaging/internal/rabbitmq"
	"websocket-messaging/internal/redis"

	"github.com/gorilla/websocket"
)

type WebSocketServer struct {
	serverID  string
	clients   sync.Map
	redis     *redis.RedisClient
	upgrader  websocket.Upgrader
	queueName string
	queue     *rabbitmq.RabbitMQ
	mutex     sync.Mutex
}

func getChannelName(userId, serverId string) string {
	return userId + "_" + serverId
}

func getMacAddress() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		if iface.HardwareAddr != nil { // Check if the interface has a MAC address
			return iface.HardwareAddr.String(), nil
		}
	}
	return "", fmt.Errorf("Mac Address not found")
}

func NewWebSocketServer(redis *redis.RedisClient, queueService *rabbitmq.RabbitMQ) *WebSocketServer {
	macAdd, err := getMacAddress()
	if err != nil {
		log.Println("Fail to get mac address: ", err)
		return nil
	}
	return &WebSocketServer{
		serverID: macAdd,
		clients:  sync.Map{},
		redis:    redis,
		queue:    queueService,
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
	ws.clients.Store(userID, conn)
	ws.redis.SetUserServer(userID, ws.serverID)

	go ws.readMessages(userID, conn)
	go ws.ReceiveMessagesRedis(userID)
}

// Receive message from consumer via channels
func (ws *WebSocketServer) ReceiveMessagesRedis(userId string) {
	ctx := context.Background()
	pubsub := ws.redis.Client.Subscribe(ctx, ws.serverID)
	defer pubsub.Close()
	ws.redis.Subscribe(pubsub, func(s string) {
		var msg models.Message
		err := json.Unmarshal([]byte(s), &msg)
		if err != nil {
			log.Println("Fail to unmarshal pubsub data", err)
			return
		}
		fmt.Println(msg)
		ws.SendMessage(msg.ReceiverID, s)
	})
}

func (ws *WebSocketServer) readMessages(userID string, conn *websocket.Conn) {
	defer conn.Close()
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading message:", err)
			ws.clients.Delete(userID)
			return
		}
		go func() {
			err := ws.queue.Publish(ws.queueName, string(message))
			if err != nil {
				log.Println("Error in publishing message into the queue :", err)
			}
		}()
		log.Printf("Received message from %s: %s", userID, string(message))
	}
}

func (ws *WebSocketServer) SendMessage(userID, message string) {
	ws.mutex.Lock() // Lock before writing
	defer ws.mutex.Unlock()
	if conn, exists := ws.clients.Load(userID); exists {
		err := conn.(*websocket.Conn).WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			log.Println("Error sending message:", err)
		}
	}
}

func (ws *WebSocketServer) Start(port, queueName string) {
	ws.queueName = queueName
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
	ctx := context.Background()
	pubsub := ws.redis.Client.Subscribe(ctx, channel)
	ws.redis.Subscribe(pubsub, func(s string) {})
}
