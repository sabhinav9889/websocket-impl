package websocket

import (
	"encoding/json"
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
		client.Mu.Lock()
		err := client.Conn.WriteMessage(websocket.TextMessage, []byte(message))
		client.Mu.Unlock()
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
		messageList.SenderID = client.UserID
		if messageList.Type == "group" {
			if messageList.Status == "register" {
				for _, receiverId := range messageList.ReceiverList {
					group := RegisterRequest{GroupID: messageList.GroupID, UserID: receiverId}
					ws.hub.Register <- &group
				}
			} else if messageList.Status == "unregister" {
				for _, receiverId := range messageList.ReceiverList {
					group := RegisterRequest{GroupID: messageList.GroupID, UserID: receiverId}
					ws.hub.Unregister <- &group
				}
			} else if messageList.Status == "broadcast" || messageList.Status == "edited" {
				msg, _ := json.Marshal(messageList)
				ws.hub.Broadcast <- string(msg)
			} else if messageList.Status == "create" {
				ws.hub.CreateGroup(messageList.GroupID, messageList.GroupName, client.UserID, messageList.ReceiverList)
			}
			continue
		}
		if len(messageList.ReceiverList) != 0 {
			for _, receiverId := range messageList.ReceiverList {
				receiverMessage := getRecieverMessage(messageList, receiverId)
				msg, err := json.Marshal(receiverMessage)
				if err != nil {
					log.WithError(err).Error("Error while marshelling message")
				}
				go ws.PublishMessage(string(msg))
			}
		} else {
			receiverMessage := getRecieverMessage(messageList, messageList.ReceiverID)
			msg, err := json.Marshal(receiverMessage)
			if err != nil {
				log.WithError(err).Error("Error while marshelling message")
			}
			go ws.PublishMessage(string(msg))
		}
		log.Info("Received message from ", client.UserID, ": to", string(message))
	}
}
