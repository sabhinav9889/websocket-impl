// server.go
package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"websocket-messaging/internal/database"
	"websocket-messaging/internal/models"
	"websocket-messaging/internal/rabbitmq"
	"websocket-messaging/internal/redis"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

type WebSocketServer struct {
	serverID  string
	clients   sync.Map // userID -> *WsClient
	redis     *redis.RedisClient
	upgrader  websocket.Upgrader
	queueName string
	queue     *rabbitmq.RabbitMQ
	hub       *Hub
	MongoDb   *database.MongoDB
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

func NewWebSocketServer(redis *redis.RedisClient, queueService *rabbitmq.RabbitMQ, dbUri, dbName, dbCollection string) *WebSocketServer {
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
		MongoDb: database.NewDatabase(dbUri, dbName, dbCollection),
	}
}

func refiningStruct(msg []string) []string {
	var messages []string
	for _, message := range msg {
		fmt.Println("Printed message is : ", message)
		var temp database.ChatMessage
		err := json.Unmarshal([]byte(message), &temp)
		if err != nil {
			log.WithError(err).Error("Failed to unmarshal message")
			continue
		}
		messagetemp := models.Message{
			MessageId:        temp.MessageId,
			Content:          temp.MessageBody,
			Status:           temp.Status,
			ReceivedAt:       temp.ReceivedAt,
			CreatedAt:        temp.CreatedAt,
			ConversationID:   temp.ConversationID,
			ReplyToMessageID: temp.ReplyToMessageID,
			SenderID:         temp.SenderID,
			ReceiverID:       temp.ReceiverID,
			GroupID:          temp.GroupID,
			GroupName:        temp.GroupName,
			Type:             temp.MessageType,
		}
		content, err := json.Marshal(messagetemp)
		messages = append(messages, string(content))
	}
	return messages
}

func (ws *WebSocketServer) RetrivePendingMessage(userID string, client *WsClient) {
	go func() {
		msg, err := ws.MongoDb.GetPendingMessages(userID)

		if err != nil {
			log.WithError(err).Error("Failed to retrieve pending messages")
			return
		}
		err = ws.MongoDb.ChangeMessageStatus(userID)
		if err != nil {
			log.WithError(err).Error("Failed to change the message status")
			return
		}
		messsages := refiningStruct(msg)
		for _, message := range messsages {
			client.Message <- message
		}
	}()
}

func (ws *WebSocketServer) PublishMessage(msg string) error {
	err := ws.queue.Publish(ws.queueName, msg)
	if err != nil {
		return err
	}
	return nil
}

func (ws *WebSocketServer) HandleConnection(w http.ResponseWriter, r *http.Request) {
	Conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.WithError(err).Error("WebSocket upgrade failed")
		return
	}
	userID := r.URL.Query().Get("userID")
	client := &WsClient{Conn: Conn, Message: make(chan string), UserID: userID}
	ws.clients.Store(userID, client)
	ws.redis.SetUserServer(userID, ws.serverID)
	ws.RetrivePendingMessage(userID, client)
	go client.readMessages(ws)
	go client.StartWriter()
}

func (ws *WebSocketServer) StartHubServer() {
	hub := NewHub()
	ws.hub = hub
	go ws.hub.Run(ws)
}

func (ws *WebSocketServer) StartRedisMessageListener() {
	ws.redis.Subscribe(ws.serverID, func(s string) {
		var msg models.Message
		if err := json.Unmarshal([]byte(s), &msg); err != nil {
			log.WithError(err).Error("Failed to unmarshal pubsub data")
			return
		}
		ws.SendMessage(msg.ReceiverID, s)
	})
}

func (ws *WebSocketServer) SendMessage(userID, message string) {
	if c, ok := ws.clients.Load(userID); ok {
		client := c.(*WsClient)
		client.Mu.Lock()
		defer client.Mu.Unlock()
		client.Message <- message
	}
}

func (ws *WebSocketServer) Start(port, queueName string) error {
	ws.queueName = queueName
	http.HandleFunc("/ws", ws.HandleConnection)
	log.WithField("port", port).Info("WebSocket Server starting")
	return http.ListenAndServe(":"+port, nil)
}

func (ws *WebSocketServer) StartHeartbeat(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ws.redis.SetWithTTL("server_heartbeat:"+ws.serverID, "alive", 10*time.Second)
		}
	}
}
