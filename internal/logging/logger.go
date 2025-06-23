package logging

import (
	"fmt"
	"sync"
	"time"
)

type LogLevel string

const (
	DEBUG LogLevel = "debug"
	INFO  LogLevel = "info"
	WARN  LogLevel = "warn"
	ERROR LogLevel = "error"
)

type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	Source    string                 `json:"source,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

type Logger struct {
	mu sync.RWMutex
}

var globalLogger *Logger
var once sync.Once

func InitializeLogger(logDir string) error {
	once.Do(func() {
		globalLogger = &Logger{}
	})
	return nil
}

func GetLogger() *Logger {
	if globalLogger == nil {
		// Initialize with default directory if not initialized
		InitializeLogger("./logs")
	}
	return globalLogger
}



func (l *Logger) Log(level LogLevel, message string, source string, data map[string]interface{}) {
	// Only log timing information to console
	if source == "event" && data != nil {
		if duration, ok := data["durationMs"]; ok {
			// Skip logging events with 0ms duration
			if durationMs, ok := duration.(int64); ok && durationMs <= 0 {
				return
			}
			if eventType, ok := data["type"]; ok {
				collection := data["collection"]
				runtime := data["runtime"]
				if level == ERROR {
					fmt.Printf("[%s] ❌ %s event on %s (%s runtime) - %vms - ERROR: %v\n", 
						time.Now().Format("15:04:05"), eventType, collection, runtime, duration, data["error"])
				} else {
					fmt.Printf("[%s] ✅ %s event on %s (%s runtime) - %vms\n", 
						time.Now().Format("15:04:05"), eventType, collection, runtime, duration)
				}
			}
		}
	}
}

func (l *Logger) Debug(message string, source string, data map[string]interface{}) {
	l.Log(DEBUG, message, source, data)
}

func (l *Logger) Info(message string, source string, data map[string]interface{}) {
	l.Log(INFO, message, source, data)
}

func (l *Logger) Warn(message string, source string, data map[string]interface{}) {
	l.Log(WARN, message, source, data)
}

func (l *Logger) Error(message string, source string, data map[string]interface{}) {
	l.Log(ERROR, message, source, data)
}

func (l *Logger) GetLogFiles() ([]string, error) {
	return []string{}, nil
}

func (l *Logger) ReadLogs(filename string, level LogLevel) ([]LogEntry, error) {
	return []LogEntry{}, nil
}

func (l *Logger) GetLogPath(filename string) string {
	return ""
}


// Global convenience functions
func Debug(message string, source string, data map[string]interface{}) {
	GetLogger().Debug(message, source, data)
}

func Info(message string, source string, data map[string]interface{}) {
	GetLogger().Info(message, source, data)
}

func Warn(message string, source string, data map[string]interface{}) {
	GetLogger().Warn(message, source, data)
}

func Error(message string, source string, data map[string]interface{}) {
	GetLogger().Error(message, source, data)
}