package websocket

import (
	"encoding/json"
	"sync"
	"websocket-messaging/internal/models"

	log "github.com/sirupsen/logrus"
)

type Group struct {
	GroupID string          `json:"group_id"`
	Name    string          `json:"name"`
	Clients map[string]bool `json:"clients"` // list of clients
	Mu      sync.RWMutex
}

type RegisterRequest struct {
	GroupID string
	UserID  string
}

type Hub struct {
	Groups     sync.Map
	Register   chan *RegisterRequest
	Unregister chan *RegisterRequest
	Broadcast  chan string
}

type Message struct {
	MessageID string `json:"message_id"`
	Content   string `json:"content"`
	GroupID   string `json:"group_id"`
	UserID    string `json:"user_id"`
}

func NewHub() *Hub {
	return &Hub{
		Groups:     sync.Map{},
		Register:   make(chan *RegisterRequest),
		Unregister: make(chan *RegisterRequest),
		Broadcast:  make(chan string),
	}
}

func (hub *Hub) PrintGroup() {
	hub.Groups.Range(func(key, value interface{}) bool {
		group := value.(*Group)
		log.Info("Group ID: ", group.GroupID, " Name: ", group.Name, key)
		return true
	})
}

func (hub *Hub) CreateGroup(groupId, name, userId string, users []string) {
	grp := Group{GroupID: groupId, Name: name, Clients: make(map[string]bool)}
	if _, ok := hub.Groups.Load(grp.GroupID); !ok {
		hub.Groups.Store(grp.GroupID, &grp)
		hub.Register <- &RegisterRequest{GroupID: grp.GroupID, UserID: userId}
		for _, user := range users {
			hub.Register <- &RegisterRequest{GroupID: grp.GroupID, UserID: user}
		}
		log.Info("Group created successfully: ", grp.Name)
	} else {
		log.Info("Group already exists: ", grp.Name)
	}
}

func (hub *Hub) Run(ws *WebSocketServer) {
	log.Info("Hub is running")
	hub.PrintGroup()
	for {
		select {
		case c1 := <-hub.Register:
			if grp, ok := hub.Groups.Load(c1.GroupID); ok {
				group := grp.(*Group)
				group.Mu.RLock()
				_, ok := group.Clients[c1.UserID]
				group.Mu.RUnlock()
				if !ok {
					group.Mu.Lock()
					group.Clients[c1.UserID] = true
					group.Mu.Unlock()
					log.Info("Client registered successfully in group")
				} else {
					log.Info("Client already registered in group")
				}
			} else {
				log.Info("Group not found")
			}
		case c2 := <-hub.Unregister:
			if grp, ok := hub.Groups.Load(c2.GroupID); ok {
				group := grp.(*Group)
				group.Mu.RLock()
				_, ok := group.Clients[c2.UserID]
				group.Mu.RUnlock()
				if ok {
					delete(group.Clients, c2.UserID)
					log.Info("Unregister user successfully : ", c2.UserID)
				} else {
					log.Info("Client not registered in group")
				}
			} else {
				log.Info("Group not found")
			}
		case c3 := <-hub.Broadcast:
			var msg models.Message
			err := json.Unmarshal([]byte(c3), &msg)
			if err != nil {
				log.WithError(err).Error("Error while broadcasting message")
				continue
			}
			if grp, ok := hub.Groups.Load(msg.GroupID); ok {
				group := grp.(*Group)
				_, ok := group.Clients[msg.SenderID]
				if ok {
					for userId, _ := range group.Clients {
						message := msg
						message.ReceiverID = userId
						messageByte, err := json.Marshal(message)
						if err != nil {
							log.WithError(err).Error("Error while broadcasting message")
							continue
						}
						ws.PublishMessage(string(messageByte))
						log.Info("Message broadcast successfully in group")
					}
				} else {
					log.Info("Client is not registered yet in the group: ", msg.SenderID)
				}
			} else {
				log.Info("Group not found")
			}
		}
	}
}
