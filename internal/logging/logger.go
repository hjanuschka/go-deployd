package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	mu       sync.RWMutex
	logDir   string
	logFile  *os.File
	filename string
}

var globalLogger *Logger
var once sync.Once

func InitializeLogger(logDir string) error {
	var err error
	once.Do(func() {
		globalLogger = &Logger{
			logDir: logDir,
		}
		err = globalLogger.setupLogFile()
	})
	return err
}

func GetLogger() *Logger {
	if globalLogger == nil {
		// Initialize with default directory if not initialized
		InitializeLogger("./logs")
	}
	return globalLogger
}

func (l *Logger) setupLogFile() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Create logs directory if it doesn't exist
	if err := os.MkdirAll(l.logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Generate filename with current date
	now := time.Now()
	l.filename = fmt.Sprintf("deployd-%s.jsonl", now.Format("2006-01-02"))
	logPath := filepath.Join(l.logDir, l.filename)

	// Close existing file if open
	if l.logFile != nil {
		l.logFile.Close()
	}

	// Open or create log file
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	l.logFile = file
	return nil
}

func (l *Logger) writeLog(entry LogEntry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if we need to rotate the log file (new day)
	now := time.Now()
	expectedFilename := fmt.Sprintf("deployd-%s.jsonl", now.Format("2006-01-02"))
	if l.filename != expectedFilename {
		if err := l.setupLogFile(); err != nil {
			return err
		}
	}

	// Write log entry as JSONL
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	_, err = l.logFile.WriteString(string(data) + "\n")
	if err == nil {
		l.logFile.Sync() // Ensure it's written to disk
	}
	return err
}

func (l *Logger) Log(level LogLevel, message string, source string, data map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Source:    source,
		Data:      data,
	}

	if err := l.writeLog(entry); err != nil {
		// Fallback to stderr if logging fails
		fmt.Fprintf(os.Stderr, "Logging error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Failed to log: %s\n", message)
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
	l.mu.RLock()
	defer l.mu.RUnlock()

	files, err := os.ReadDir(l.logDir)
	if err != nil {
		return nil, err
	}

	var logFiles []string
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".jsonl" {
			logFiles = append(logFiles, file.Name())
		}
	}

	return logFiles, nil
}

func (l *Logger) ReadLogs(filename string, level LogLevel) ([]LogEntry, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if filename == "" || filename == "current" {
		filename = l.filename
	}

	logPath := filepath.Join(l.logDir, filename)
	data, err := os.ReadFile(logPath)
	if err != nil {
		return nil, err
	}

	var logs []LogEntry
	lines := string(data)
	for _, line := range splitLines(lines) {
		if line == "" {
			continue
		}

		var entry LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // Skip malformed lines
		}

		// Filter by level if specified
		if level != "" && entry.Level != level {
			continue
		}

		logs = append(logs, entry)
	}

	return logs, nil
}

func (l *Logger) GetLogPath(filename string) string {
	if filename == "" || filename == "current" {
		filename = l.filename
	}
	return filepath.Join(l.logDir, filename)
}

func splitLines(s string) []string {
	var lines []string
	var line string
	for _, r := range s {
		if r == '\n' {
			lines = append(lines, line)
			line = ""
		} else {
			line += string(r)
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
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