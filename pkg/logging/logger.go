package logging

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

func InitLogger(format string, level logrus.Level) *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(level)

	if format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{})
	}

	return logger
}

// Logger handles structured logging
type Logger struct {
	stdLog  *log.Logger
	service string
	traceID string
	groupID string
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Additional interface{} `json:"additional,omitempty"`
	Timestamp  string      `json:"timestamp"`
	Level      string      `json:"level"`
	Service    string      `json:"service"`
	TraceID    string      `json:"trace_id,omitempty"`
	GroupID    string      `json:"group_id,omitempty"`
	Message    string      `json:"message"`
}

// NewLogger creates a new logger for a service
func NewLogger(service string) *Logger {
	return &Logger{
		service: service,
		stdLog:  log.New(os.Stdout, "", 0),
	}
}

// WithTraceID adds trace ID to the logger
func (l *Logger) WithTraceID(traceID string) *Logger {
	return &Logger{
		service: l.service,
		traceID: traceID,
		groupID: l.groupID,
		stdLog:  l.stdLog,
	}
}

// WithGroupID adds group ID to the logger
func (l *Logger) WithGroupID(groupID string) *Logger {
	return &Logger{
		service: l.service,
		traceID: l.traceID,
		groupID: groupID,
		stdLog:  l.stdLog,
	}
}

// GetTraceID returns the trace ID
func (l *Logger) GetTraceID() string {
	return l.traceID
}

// GetGroupID returns the group ID
func (l *Logger) GetGroupID() string {
	return l.groupID
}

// Info logs an info message
func (l *Logger) Info(msg string, additional interface{}) {
	l.log("INFO", msg, additional)
}

// Error logs an error message
func (l *Logger) Error(msg string, additional interface{}) {
	l.log("ERROR", msg, additional)
}

// log handles the actual logging
func (l *Logger) log(level, msg string, additional interface{}) {
	entry := LogEntry{
		Timestamp:  time.Now().Format(time.RFC3339),
		Level:      level,
		Service:    l.service,
		TraceID:    l.traceID,
		GroupID:    l.groupID,
		Message:    msg,
		Additional: additional,
	}

	entryJSON, err := json.Marshal(entry)
	if err != nil {
		l.stdLog.Printf("Error marshaling log entry: %v", err)
		return
	}

	l.stdLog.Println(string(entryJSON))
}
