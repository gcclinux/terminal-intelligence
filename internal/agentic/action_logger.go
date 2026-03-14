package agentic

import (
	"fmt"
	"time"
)

// ActionLogger writes timestamped log entries to the AI chat pane
// via a notify callback (wrapping AIChatPane.DisplayNotification).
type ActionLogger struct {
	notify func(string)
}

// NewActionLogger creates an ActionLogger that delegates to the given notify function.
func NewActionLogger(notify func(string)) *ActionLogger {
	return &ActionLogger{notify: notify}
}

// Log formats a message with a "HH:MM:SS" timestamp prefix and sends it
// through the notify callback.
func (al *ActionLogger) Log(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	stamped := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg)
	al.notify(stamped)
}
