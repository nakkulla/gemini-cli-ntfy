package notification

import (
	"fmt"
	"os"
)

// StdoutNotifier prints notifications to stdout (for testing/debugging)
type StdoutNotifier struct{}

// NewStdoutNotifier creates a new stdout notifier
func NewStdoutNotifier() *StdoutNotifier {
	return &StdoutNotifier{}
}

// Send implements the Notifier interface
func (s *StdoutNotifier) Send(notification Notification) error {
	fmt.Fprintf(os.Stderr, "[NOTIFY] %s: %s\n", notification.Title, notification.Message)
	return nil
}