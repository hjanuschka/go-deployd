package logging

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type LogLevel string

const (
	DEBUG          LogLevel = "debug"
	INFO           LogLevel = "info"
	WARN           LogLevel = "warn"
	ERROR          LogLevel = "error"
	USER_GENERATED LogLevel = "user-generated"
)

type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	Source    string                 `json:"source,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

type Logger struct {
	mu         sync.RWMutex
	logDir     string
	file       *os.File
	logChan    chan *LogEntry
	stopChan   chan struct{}
	wg         sync.WaitGroup
	devMode    bool
}

var globalLogger *Logger
var once sync.Once

func InitializeLogger(logDir string) error {
	return InitializeLoggerWithDevMode(logDir, false)
}

func InitializeLoggerWithDevMode(logDir string, devMode bool) error {
	once.Do(func() {
		chanSize := 100 // Buffer size for channel
		if devMode {
			chanSize = 1000 // Larger channel buffer for dev mode
		}
		
		globalLogger = &Logger{
			logDir:   logDir,
			logChan:  make(chan *LogEntry, chanSize),
			stopChan: make(chan struct{}),
			devMode:  devMode,
		}

		// Create log directory if it doesn't exist
		if err := os.MkdirAll(logDir, 0755); err != nil {
			fmt.Printf("Warning: Failed to create log directory: %v\n", err)
			return
		}

		// Open today's log file
		if err := globalLogger.openLogFile(); err != nil {
			fmt.Printf("Warning: Failed to open log file: %v\n", err)
		}
		
		// Start the async logging goroutine
		globalLogger.wg.Add(1)
		go globalLogger.logWorker()
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

// openLogFile opens or creates today's log file
func (l *Logger) openLogFile() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Close existing file if open
	if l.file != nil {
		l.file.Close()
	}

	// Create filename with today's date
	today := time.Now().Format("2006-01-02")
	filename := filepath.Join(l.logDir, fmt.Sprintf("%s.jsonl", today))

	// Open file in append mode, create if it doesn't exist
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	l.file = file
	return nil
}

func (l *Logger) Log(level LogLevel, message string, source string, data map[string]interface{}) {
	// Console output for event timing
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

	// Write to JSON-L log file
	l.writeLogEntry(level, message, source, data)
}

// logWorker runs in a goroutine and handles async logging
func (l *Logger) logWorker() {
	defer l.wg.Done()
	
	ticker := time.NewTicker(100 * time.Millisecond) // Flush every 100ms in dev mode
	if !l.devMode {
		ticker = time.NewTicker(1 * time.Second) // Flush every second in production
	}
	defer ticker.Stop()
	
	var batch []*LogEntry
	batchSize := 10
	if l.devMode {
		batchSize = 50 // Larger batches in dev mode
	}
	
	for {
		select {
		case entry := <-l.logChan:
			batch = append(batch, entry)
			
			// Flush batch if it's full
			if len(batch) >= batchSize {
				l.flushBatch(batch)
				batch = batch[:0] // Reset slice
			}
			
		case <-ticker.C:
			// Flush remaining entries on timer
			if len(batch) > 0 {
				l.flushBatch(batch)
				batch = batch[:0]
			}
			
		case <-l.stopChan:
			// Flush any remaining entries before stopping
			if len(batch) > 0 {
				l.flushBatch(batch)
			}
			return
		}
	}
}

func (l *Logger) flushBatch(batch []*LogEntry) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	if l.file == nil {
		return
	}
	
	// Write all entries in batch
	for _, entry := range batch {
		if jsonData, err := json.Marshal(entry); err == nil {
			l.file.WriteString(string(jsonData) + "\n")
		}
	}
	
	// Only sync once per batch for better performance
	l.file.Sync()
}

func (l *Logger) writeLogEntry(level LogLevel, message string, source string, data map[string]interface{}) {
	entry := &LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Source:    source,
		Data:      data,
	}
	
	// Try to send to channel, drop if full (non-blocking)
	select {
	case l.logChan <- entry:
		// Successfully queued
	default:
		// Channel full, drop log entry (or could implement overflow handling)
		if l.devMode {
			fmt.Printf("Warning: Log channel full, dropping entry: %s\n", message)
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

func (l *Logger) UserGenerated(message string, source string, data map[string]interface{}) {
	l.Log(USER_GENERATED, message, source, data)
}

// Shutdown gracefully stops the logger and flushes remaining entries
func (l *Logger) Shutdown() {
	if l.stopChan != nil {
		close(l.stopChan)
		l.wg.Wait() // Wait for goroutine to finish
	}
	
	l.mu.Lock()
	defer l.mu.Unlock()
	
	if l.file != nil {
		l.file.Sync()
		l.file.Close()
		l.file = nil
	}
}

// Global shutdown function
func Shutdown() {
	if globalLogger != nil {
		globalLogger.Shutdown()
	}
}

func (l *Logger) GetLogFiles() ([]string, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.logDir == "" {
		return []string{}, nil
	}

	files, err := os.ReadDir(l.logDir)
	if err != nil {
		return []string{}, err
	}

	var logFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".jsonl") {
			logFiles = append(logFiles, file.Name())
		}
	}

	// Sort files by name (which includes date)
	sort.Strings(logFiles)
	return logFiles, nil
}

func (l *Logger) ReadLogs(filename string, level LogLevel) ([]LogEntry, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var logFile string
	if filename == "" {
		// Use today's log file if no filename specified
		today := time.Now().Format("2006-01-02")
		logFile = filepath.Join(l.logDir, fmt.Sprintf("%s.jsonl", today))
	} else {
		logFile = filepath.Join(l.logDir, filename)
	}

	file, err := os.Open(logFile)
	if err != nil {
		return []LogEntry{}, nil // Return empty if file doesn't exist
	}
	defer file.Close()

	var logs []LogEntry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var entry LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // Skip invalid JSON lines
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
	if filename == "" {
		today := time.Now().Format("2006-01-02")
		filename = fmt.Sprintf("%s.jsonl", today)
	}
	return filepath.Join(l.logDir, filename)
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

func UserGenerated(message string, source string, data map[string]interface{}) {
	GetLogger().UserGenerated(message, source, data)
}
