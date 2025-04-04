// interface.go
package database

type DatabaseInterface interface {
    SaveMessage(userID string, message string) error
    GetPendingMessages(userID string) ([]string, error)
    DeletePendingMessages(userID string) error
}