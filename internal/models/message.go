package models

type MessageStruct struct {
	MessageId    string   `json:"message_id"`
	Type         string   `json:"type"`               // Message type (e.g., "status", "content", "file")
	Content      string   `json:"content"`            // The message content (e.g., status message or text)
	Status       string   `json:"status,omitempty"`   // Optional: status of the user (e.g., "online", "typing")
	FileName     string   `json:"filename,omitempty"` // Optional: name of the file
	FileType     string   `json:"filetype,omitempty"` // Optional: MIME type of the file (e.g., "application/pdf")
	FileData     []byte   `json:"filedata,omitempty"` // Optional: file data (binary content)
	ReceiverList []string `json:"receiver_lis"`       // Receiver who receive this message from sender
	SenderID     string   `json:"sender_id"`          // sender id can send messages to the multiple users
	TimeStamp    int64    `json:"timestamp"`          // timestamp of message received
}

type Message struct {
	MessageId  string `json:"message_id"`
	Status     string `json:"status"`      // edit or not
	Type       string `json:"type"`        // Message type (e.g., "status", "content", "file")
	Content    string `json:"content"`     // The message content (e.g., status message or text)
	ReceiverID string `json:"receiver_id"` // Receiver who receive this message from sender
	SenderID   string `json:"sender_id"`   // sender id can send messages to the multiple users
	TimeStamp  int64  `json:"timestamp"`   // timestamp of message received
}
