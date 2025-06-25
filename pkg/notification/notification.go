package notification

import "time"

// Notification represents a notification to be sent
type Notification struct {
	Title   string
	Message string
	Time    time.Time
	Pattern string
}

// Notifier interface for sending notifications
type Notifier interface {
	Send(notification Notification) error
}