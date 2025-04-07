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

	"websocket-messaging/internal/models"
	"websocket-messaging/internal/rabbitmq"
	"websocket-messaging/internal/redis"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

type wsClient struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

type WebSocketServer struct {
	serverID  string
	clients   sync.Map // userID -> *wsClient
	redis     *redis.RedisClient
	upgrader  websocket.Upgrader
	queueName string
	queue     *rabbitmq.RabbitMQ
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
		if iface.HardwareAddr != nil {
			return iface.HardwareAddr.String(), nil
		}
	}
	return "", fmt.Errorf("mac address not found")
}

func NewWebSocketServer(redis *redis.RedisClient, queueService *rabbitmq.RabbitMQ) *WebSocketServer {
	macAdd, err := getMacAddress()
	if err != nil {
		log.WithError(err).Error("Failed to get MAC address")
		return nil
	}
	return &WebSocketServer{
		serverID: macAdd,
		redis:    redis,
		queue:    queueService,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // TODO: tighten this in production
			},
		},
	}
}

func (ws *WebSocketServer) HandleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.WithError(err).Error("WebSocket upgrade failed")
		return
	}

	userID := r.URL.Query().Get("userID")
	client := &wsClient{conn: conn}
	ws.clients.Store(userID, client)
	ws.redis.SetUserServer(userID, ws.serverID)

	go ws.readMessages(userID, client)
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

func (ws *WebSocketServer) readMessages(userID string, client *wsClient) {
	defer client.conn.Close()
	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			log.WithFields(log.Fields{"userID": userID, "error": err}).Error("WebSocket read error")
			ws.clients.Delete(userID)
			return
		}

		log.WithField("userID", userID).Infof("Received message: %s", string(message))

		go func(msg string) {
			if err := ws.queue.Publish(ws.queueName, msg); err != nil {
				log.WithError(err).Error("Failed to publish message to queue")
			}
		}(string(message))
	}
}

func (ws *WebSocketServer) SendMessage(userID, message string) {
	if c, ok := ws.clients.Load(userID); ok {
		client := c.(*wsClient)
		client.mu.Lock()
		err := client.conn.WriteMessage(websocket.TextMessage, []byte(message))
		client.mu.Unlock()
		if err != nil {
			log.WithError(err).Error("Failed to send WebSocket message")
		}
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
