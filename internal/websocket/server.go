// server.go
package websocket

import (
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
	return "", fmt.Errorf("mac address not found")
}

func getRecieverMessage(messageData models.MessageStruct, receiverId string) models.Message {
	var message models.Message
	if messageData.Type == "content" {
		message.Content = messageData.Content
	}
	message.MessageId = messageData.MessageId
	message.Status = messageData.Status
	message.Type = messageData.Type
	message.ReceiverID = receiverId
	message.SenderID = messageData.SenderID
	message.TimeStamp = messageData.TimeStamp
	return message
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
}

func (ws *WebSocketServer) StartRedisMessageListener() {
	ws.redis.Subscribe(ws.serverID, func(s string) {
		var msg models.Message
		err := json.Unmarshal([]byte(s), &msg)
		if err != nil {
			log.Println("Fail to unmarshal pubsub data", err)
			return
		}
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
		var messageList models.MessageStruct
		err = json.Unmarshal(message, &messageList)
		if err != nil {
			log.Println("Error while unmarshelling message", err)
			return
		}
		for _, receiverId := range messageList.ReceiverList {
			receiverMessage := getRecieverMessage(messageList, receiverId)
			msg, err := json.Marshal(receiverMessage)
			if err != nil {
				log.Println("Error while marshelling message", err)
			}
			go func() {
				err := ws.queue.Publish(ws.queueName, string(msg))
				if err != nil {
					log.Println("Error in publishing message into the queue :", err)
				}
			}()
		}
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
