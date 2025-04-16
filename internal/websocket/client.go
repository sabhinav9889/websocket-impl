package websocket

import (
	"encoding/json"
	"fmt"
	"sync"
	"websocket-messaging/internal/models"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

type WsClient struct {
	UserID  string
	Conn    *websocket.Conn
	Mu      sync.Mutex
	Message chan string
}

func getRecieverMessage(messageData models.MessageStruct, receiverId string) models.Message {
	var message models.Message
	if messageData.Type != "status" {
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

func (client *WsClient) StartWriter() {
	for message := range client.Message {
		err := client.Conn.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			log.WithError(err).Error("Failed to send WebSocket message")
			close(client.Message) // this will break the range loop
			return
		}
	}
}

func (client *WsClient) readMessages(ws *WebSocketServer) {
	Conn := client.Conn
	defer Conn.Close()
	for {
		_, message, err := Conn.ReadMessage()
		if err != nil {
			log.WithError(err).Error("Error reading message")
			ws.clients.Delete(client.UserID)
			ws.redis.DeleteUserServer(client.UserID)
			log.Info("Client disconnected: ", client.UserID)
			return
		}
		var messageList models.MessageStruct
		err = json.Unmarshal(message, &messageList)
		if err != nil {
			log.WithError(err).Error("Error while unmarshelling message")
			continue
		}
		if messageList.Type == "group" {
			group := RegisterRequest{GroupID: messageList.GroupID, UserID: messageList.SenderID}
			if messageList.Status == "register" {
				log.Info("Received message from ", client.UserID, ": to", string(message), group)
				ws.hub.Register <- &group
			} else if messageList.Status == "unregister" {
				ws.hub.Unregister <- &group
			} else if messageList.Status == "broadcast" {
				fmt.Println("broadcast")
				ws.hub.Broadcast <- string(message)
			} else if messageList.Status == "create" {
				ws.hub.CreateGroup(messageList.GroupID, messageList.GroupName)
			}
			continue
		}
		for _, receiverId := range messageList.ReceiverList {
			receiverMessage := getRecieverMessage(messageList, receiverId)
			msg, err := json.Marshal(receiverMessage)
			if err != nil {
				log.WithError(err).Error("Error while marshelling message")
			}
			go ws.PublishMessage(string(msg))
		}
		log.Info("Received message from ", client.UserID, ": to", string(message))
	}
}
