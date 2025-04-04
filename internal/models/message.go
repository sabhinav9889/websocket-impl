package models

type Message struct {
	MessageID string `json:"message_id"`

	SenderID   string `json:"sender_id"`
	ReceiverID string `json:"receiver_id"`
	Content    string `json:"content"`
}
